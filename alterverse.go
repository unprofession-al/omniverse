package main

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"sync"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type alterverseConfig map[string]map[string]string

func (ac alterverseConfig) GetAlterverse(name string, files map[string][]byte) (alterverse, error) {
	definitions, ok := ac[name]
	if !ok {
		return alterverse{}, fmt.Errorf("alterverse definitions for '%s' not found", name)
	}
	a := NewAlterverse(definitions, files)
	return a, nil
}

type alterverse struct {
	sync.RWMutex
	definitions map[string]string
	files       map[string][]byte
}

func NewAlterverse(definitions map[string]string, files map[string][]byte) alterverse {
	return alterverse{
		definitions: definitions,
		files:       files,
	}
}

func (a alterverse) Definitions() map[string]string {
	return a.definitions
}

func (a alterverse) SubstituteDefinitions(expressionTemplate string) (map[string][]byte, error) {
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
				log.Info(fmt.Sprintf("Change on line %d of file %s:\n\told: %s\n\tnew: %s", linenum, path, string(line), string(newLine)))
			}
			out = append(out, newLine...)
			out = append(out, lb...)
			linenum++
		}

		if err := scanner.Err(); err != nil {
			return rendered, fmt.Errorf("failed to substitute strings in altiverse file '%s', error was %s", path, err.Error())
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
	values := byLen{}
	for k, v := range a.definitions {
		reverseDefinitions[v] = k
		values = append(values, v)
	}
	sort.Sort(sort.Reverse(byLen(values)))

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

// byLen implements the sort.Interface and allows to sort an array of strings
// by its length. See https://golang.org/pkg/sort/#Interface
type byLen []string

// Len is part of the sort.Interface
func (a byLen) Len() int {
	return len(a)
}

// Less is part of the sort.Interface
func (a byLen) Less(i, j int) bool {
	return len(a[i]) < len(a[j])
}

// Swap is part of the sort.Interface
func (a byLen) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
