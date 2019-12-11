package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type syncer struct {
	sync.RWMutex
	basedir string
	ignore  []string
	log     chan string
}

func NewSyncer(basedir string, ignored []string, log chan string) (*syncer, error) {
	abs, err := filepath.Abs(basedir)
	if err != nil {
		return nil, err
	}

	s := &syncer{
		basedir: abs,
		ignore:  ignored,
		log:     log,
	}

	f, err := os.Stat(basedir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Base directory '%s' does not exist", basedir)
	} else if !f.IsDir() {
		return nil, fmt.Errorf("Base directory '%s' seems to be a file", basedir)
	} else if err != nil {
		return nil, err
	}

	return s, nil
}

func (s syncer) listFiles() ([]string, error) {
	list := []string{}

	err := filepath.Walk(s.basedir, func(path string, info os.FileInfo, err error) error {
		path = strings.TrimPrefix(path, s.basedir)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimPrefix(path, "\\")

		if info.IsDir() || s.isIgnored(path) {
			return nil
		}

		list = append(list, path)
		return nil
	})
	return list, err
}

func (s syncer) isIgnored(path string) bool {
	out := false
	for _, pattern := range s.ignore {
		matched, err := filepath.Match(pattern, path)
		if matched || err != nil {
			out = true
		}
	}
	return out
}

func (s syncer) deleteFiles(del []string) error {
	for _, file := range del {
		if s.isIgnored(file) {
			continue
		}
		path := filepath.Join(s.basedir, file)
		err := os.Remove(path)
		if err != nil {
			return err
		}
		s.log <- fmt.Sprintf("File '%s' deleted", path)
	}
	return nil
}

func (s syncer) WriteFiles(files map[string][]byte, del bool) error {
	s.Lock()
	defer s.Unlock()

	if del {
		list, err := s.listFiles()
		if err != nil {
			return err
		}

		obsolete := []string{}
		for _, f := range list {
			if _, ok := files[f]; !ok {
				obsolete = append(obsolete, f)
			}
		}
		if len(obsolete) > 0 {
			s.log <- fmt.Sprintf("The following files are not present in singularity and will be therefore removed: %s", strings.Join(obsolete, ", "))
			s.deleteFiles(obsolete)
		}
	}

	for name, data := range files {
		err := s.writeFile(name, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s syncer) writeFile(name string, data []byte) error {
	if s.isIgnored(name) {
		return nil
	}

	path := filepath.Join(s.basedir, name)
	s.log <- fmt.Sprintf("Writing file '%s'...", path)

	dir := filepath.Dir(path)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	var file *os.File
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		file, err = os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
	} else if err != nil {
		return err
	} else {
		file, err = os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (s syncer) ReadFiles() (map[string][]byte, error) {
	s.Lock()
	defer s.Unlock()

	out := map[string][]byte{}
	err := filepath.Walk(s.basedir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		path = strings.TrimPrefix(path, s.basedir)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimPrefix(path, "\\")
		out[path] = data
		return nil
	})

	return out, err
}

func detectLineBreak(in []byte) (string, []byte) {
	lb := map[string][]byte{
		"crlf": []byte("\r\n"),
		"lfcr": []byte("\n\r"),
		"cr":   []byte("\r"),
		"lf":   []byte("\n"),
	}

	for k, v := range lb {
		if bytes.Contains(in, v) {
			return k, v
		}
	}

	return "unknown", lb["lf"]
}
