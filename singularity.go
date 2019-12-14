package main

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sync"

	log "github.com/sirupsen/logrus"
)

type singularityConfig struct {
	Expression         string `yaml:"expression" json:"expression"`
	ExpressionTemplate string `yaml:"expression_template" json:"expression_template"`
}

type singularity struct {
	sync.RWMutex       `yaml:"-" json:"-"`
	Expression         string `yaml:"expression" json:"expression"`
	ExpressionTemplate string `yaml:"expression_template" json:"expression_template"`

	files map[string][]byte           `yaml:"-" json:"-"`
	keys  map[string]map[string][]int `yaml:"-" json:"-"`
}

func NewSingularity(c singularityConfig, data map[string][]byte) (*singularity, error) {
	s := &singularity{
		Expression:         c.Expression,
		ExpressionTemplate: c.ExpressionTemplate,
		files:              data,
		keys:               map[string]map[string][]int{},
	}
	err := s.Parse()
	return s, err
}

func (s *singularity) Parse() error {
	s.RLock()
	defer s.RUnlock()

	for file, data := range s.files {
		keys, err := s.findKeysInFile(data)
		if err != nil {
			return err
		}
		s.keys[file] = keys
	}

	return nil
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
	for name := range s.files {
		for key, lines := range s.keys[name] {
			if _, ok := keys[key]; ok {
				keys[key][name] = lines
			} else {
				keys[key] = map[string][]int{name: lines}
			}
		}
	}
	return keys
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

	for path, data := range s.files {
		_, lb := detectLineBreak(data)

		if len(s.keys[path]) == 0 {
			log.Info(fmt.Sprintf("File '%s' can be simply copied, does not contain keys...", path))
			rendered[path] = data
			continue
		}
		file := bytes.NewReader(data)

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
			return rendered, fmt.Errorf("failed to substitute strings in singularity file '%s', error was %s", path, err.Error())
		}

		rendered[path] = out
	}

	return rendered, nil
}
