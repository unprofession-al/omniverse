package main

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

type singularity struct {
	sync.RWMutex       `yaml:"-" json:"-"`
	Expression         string                     `yaml:"expression" json:"expression"`
	ExpressionTemplate string                     `yaml:"expression_template" json:"expression_template"`
	files              map[string]singularityFile `yaml:"-" json:"-"`
}

type singularityFile struct {
	keys map[string][]int
	data []byte
}

func (s *singularity) Read(basepath string) error {
	s.Lock()
	s.files = map[string]singularityFile{}
	// TODO: defer unlock?
	s.Unlock()

	sy, err := NewSyncer(basepath, []string{})
	if err != nil {
		return err
	}

	files, err := sy.ReadFiles()
	if err != nil {
		return err
	}

	for file, data := range files {
		keys, err := s.findKeysInFile(data)
		if err != nil {
			return err
		}
		s.files[file] = singularityFile{keys: keys, data: data}
	}

	return nil
}

func (s singularity) Write(files map[string][]byte, dest string) error {
	s.Lock()
	defer s.Unlock()

	ignore := []string{}
	sy, err := NewSyncer(dest, ignore)
	if err != nil {
		return err
	}

	deleteObsolete := true
	return sy.WriteFiles(files, deleteObsolete)
}

func (s *singularity) findKeysInFile(data []byte) (map[string][]int, error) {
	keys := map[string][]int{}
	file := bytes.NewReader(data)

	re, err := regexp.Compile(s.Expression)
	if err != nil {
		return keys, err
	}

	scanner := bufio.NewScanner(file)
	line := 1
	for scanner.Scan() {
		sm := re.FindAllStringSubmatch(scanner.Text(), -1)
		for _, match := range sm {
			if len(match) >= 2 {
				key := match[1]
				if _, ok := keys[key]; ok {
					keys[key] = append(keys[key], line)
				} else {
					keys[key] = []int{line}
				}
			}
		}
		line++
	}

	if err := scanner.Err(); err != nil {
		return keys, err
	}
	return keys, nil
}

func (s *singularity) GetKeys() map[string]map[string][]int {
	keys := map[string]map[string][]int{}
	for name, sf := range s.files {
		for key, lines := range sf.keys {
			if _, ok := keys[key]; ok {
				keys[key][name] = lines
			} else {
				keys[key] = map[string][]int{name: lines}
			}
		}
	}
	return keys
}

func (s *singularity) CheckIfKeysDefined(definitions map[string]string) []error {
	s.RLock()
	defer s.RUnlock()

	out := []error{}
	for k, v := range s.GetKeys() {
		if _, ok := definitions[k]; !ok {
			files := []string{}
			for f, lines := range v {
				files = append(files, fmt.Sprintf("%s %v", f, lines))
			}
			err := fmt.Errorf("key '%s' present in singularity (files %s) but not defined in input", k, strings.Join(files, ", "))
			out = append(out, err)
		}
	}
	return out
}

func (s *singularity) CheckIfDefinedIsKey(definitions map[string]string) []error {
	s.RLock()
	defer s.RUnlock()

	out := []error{}
	keys := s.GetKeys()
	for k := range definitions {
		if _, ok := keys[k]; !ok {
			err := fmt.Errorf("definition of key '%s' present but key does not exist in singularity ", k)
			out = append(out, err)
		}
	}
	return out
}

func (s *singularity) GetLineReplacer(definitions map[string]string) (func([]byte) ([]byte, bool), error) {
	re, err := regexp.Compile(s.Expression)
	out := func(in []byte) ([]byte, bool) {
		changed := false
		sm := re.FindAllSubmatch(in, -1)
		for _, match := range sm {
			if len(match) >= 2 {
				in = bytes.Replace(in, match[0], []byte(definitions[string(match[1])]), 1)
				changed = true
			}
		}
		return in, changed
	}

	return out, err
}

func (s *singularity) Generate(basepath string, definitions map[string]string) (map[string][]byte, error) {
	s.RLock()
	defer s.RUnlock()

	rendered := map[string][]byte{}
	lr, err := s.GetLineReplacer(definitions)
	if err != nil {
		return rendered, err
	}

	for path, sf := range s.files {
		_, lb := detectLineBreak(sf.data)

		if len(sf.keys) == 0 {
			log.Info(fmt.Sprintf("File '%s' can be simply copied, does not contain keys...", path))
			rendered[path] = sf.data
			continue
		}
		file := bytes.NewReader(sf.data)

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
			return rendered, err
		}

		rendered[path] = out
	}

	return rendered, nil
}
