package main

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"sync"
	"text/template"
)

type alterverseConfig map[string]map[string]string

func (ac alterverseConfig) GetAlterverses() map[string]alterverse {
	out := map[string]alterverse{}
	for name, d := range ac {
		out[name] = alterverse{definitions: d, name: name}
	}
	return out
}

func (ac alterverseConfig) GetAlterverse(name string) (alterverse, error) {
	a := alterverse{}
	definitions, ok := ac[name]
	if !ok {
		return a, fmt.Errorf("alterverse definitions for '%s' not found", name)
	}
	a.definitions = definitions
	return a, nil
}

type alterverse struct {
	sync.RWMutex
	name        string
	files       map[string][]byte
	definitions map[string]string
}

func (a alterverse) Definitions() map[string]string {
	return a.definitions
}

func (a *alterverse) Read(basepath string, log chan string) error {
	a.Lock()
	defer a.Unlock()

	a.files = map[string][]byte{}

	sy, err := NewSyncer(basepath, []string{}, log)
	if err != nil {
		return err
	}

	files, err := sy.ReadFiles()
	if err != nil {
		return err
	}

	for file, data := range files {
		a.files[file] = data
	}

	return nil
}

func (a alterverse) Write(files map[string][]byte, dest string, log chan string) error {
	a.Lock()
	defer a.Unlock()

	ignore := []string{}
	s, err := NewSyncer(dest, ignore, log)
	if err != nil {
		return err
	}

	deleteObsolete := true
	return s.WriteFiles(files, deleteObsolete)
}

func (a alterverse) SubstituteDefinitions(expressionTemplate string, log chan string) (map[string][]byte, error) {
	a.RLock()
	defer a.RUnlock()

	rendered := map[string][]byte{}
	lr, err := a.GetLineReplacer(expressionTemplate)
	if err != nil {
		return rendered, err
	}

	for path, b := range a.files {
		_, lb := detectLineBreak(b)
		file := bytes.NewReader(b)

		out := []byte{}
		scanner := bufio.NewScanner(file)
		linenum := 1
		for scanner.Scan() {
			line := scanner.Bytes()
			newLine, changed := lr(line)
			if changed {
				log <- fmt.Sprintf("Change on line %d of file %s:\n\told: %s\n\tnew: %s", linenum, path, string(line), string(newLine))
			}
			out = append(out, newLine...)
			out = append(out, lb...)
			linenum++
		}

		if err := scanner.Err(); err != nil {
			return rendered, err
		}

		rendered[path] = out
	}

	return rendered, nil
}

func (a alterverse) GetLineReplacer(expressionTemplate string) (func([]byte) ([]byte, bool), error) {
	// Reverse definitions (k=v/v=k), sort definition values by length
	// This is to ensure that long strings are replaced first to avoid a
	// potential conflict with shorter strings that are substrings of the
	// larger string.
	reverseDefinitions := make(map[string]string)
	values := ByLen{}
	for k, v := range a.definitions {
		reverseDefinitions[v] = k
		values = append(values, v)
	}
	sort.Sort(sort.Reverse(ByLen(values)))

	tmpl, err := template.New("expression").Parse(expressionTemplate)
	out := func(in []byte) ([]byte, bool) {
		changed := false
		for _, v := range values {
			val := []byte(v)
			if bytes.Contains(in, val) {
				var expression bytes.Buffer
				err = tmpl.Execute(&expression, reverseDefinitions[v])
				in = bytes.ReplaceAll(in, val, expression.Bytes())
				changed = true
			}
		}
		return in, changed
	}

	return out, err
}

type ByLen []string

func (a ByLen) Len() int {
	return len(a)
}

func (a ByLen) Less(i, j int) bool {
	return len(a[i]) < len(a[j])
}

func (a ByLen) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
