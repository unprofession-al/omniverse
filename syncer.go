package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Syncer allows read and write from a certain directory
type Syncer struct {
	sync.RWMutex
	basedir string
	ignore  []string
}

// NewSyncer takes a path to its basedir, a list of ignored files as well a
// string channel for logging reasons. It returns a Syncer and and (if
// adequate) an error.
func NewSyncer(basedir string, ignored []string) (*Syncer, error) {
	abs, err := filepath.Abs(basedir)
	if err != nil {
		return nil, err
	}

	s := &Syncer{
		basedir: abs,
		ignore:  ignored,
	}

	f, err := os.Stat(basedir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("base directory '%s' does not exist", basedir)
	} else if !f.IsDir() {
		return nil, fmt.Errorf("base directory '%s' seems to be a file", basedir)
	} else if err != nil {
		return nil, err
	}

	return s, nil
}

func (s Syncer) listFiles() ([]string, error) {
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

func (s Syncer) isIgnored(path string) bool {
	for _, pattern := range s.ignore {
		matched := strings.Contains(path, pattern)
		if matched {
			return true
		}
	}
	return false
}

func (s Syncer) deleteFiles(del []string) error {
	for _, file := range del {
		if s.isIgnored(file) {
			continue
		}
		path := filepath.Join(s.basedir, file)
		err := os.Remove(path)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("File '%s' deleted", path))
	}
	return nil
}

// WriteFiles writes the files passed to the function as a map where
// the keys of the map are the relative file paths, the value is the
// actual content of the files as a byte slice.
//
// The del option configures if files that are absent in the map passed
// but present on the file system should be deleted.
func (s Syncer) WriteFiles(files map[string][]byte, del bool) error {
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
			log.Info(fmt.Sprintf("The following files are not present in singularity and will be therefore removed: %s", strings.Join(obsolete, ", ")))
			s.deleteFiles(obsolete)
		}
	}

	for name, data := range files {
		err := s.writeFile(name, data)
		if err != nil {
			return fmt.Errorf("failed writing file '%s', error is: %s", name, err.Error())
		}
	}
	return nil
}

func (s Syncer) writeFile(name string, data []byte) error {
	if s.isIgnored(name) {
		return nil
	}

	path := filepath.Join(s.basedir, name)
	log.Info(fmt.Sprintf("Writing file '%s'...", path))

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

// ReadFiles returns the files in the basedir of the Syncer as a map.
// The keys of the map are the relative file paths, the value is the
// actual content of the files as a byte slice.
func (s Syncer) ReadFiles() (map[string][]byte, error) {
	s.Lock()
	defer s.Unlock()

	out := map[string][]byte{}
	err := filepath.Walk(s.basedir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("could not read file '%s', error was: %s", path, err.Error())
		}

		if info.IsDir() || s.isIgnored(path) {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read file '%s', error was: %s", path, err.Error())
		}

		path = strings.TrimPrefix(path, s.basedir)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimPrefix(path, "\\")
		out[path] = data
		return nil
	})

	return out, err
}

// detectLineBreaks find the first occurrence af a line break and returns its
// representation
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
