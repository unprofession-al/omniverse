package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Syncer allows read and write from a certain directory
type Syncer struct {
	basedir string
	ignore  *regexp.Regexp
}

// NewSyncer takes a path to its basedir, a list of ignored files as well a
// string channel for logging reasons. It returns a Syncer and and (if
// adequate) an error.
func NewSyncer(basedir, ignored string) (*Syncer, error) {
	abs, err := filepath.Abs(basedir)
	if err != nil {
		return nil, err
	}
	f, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("base directory '%s' does not exist", basedir)
	} else if !f.IsDir() {
		return nil, fmt.Errorf("base directory '%s' seems to be a file", basedir)
	} else if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(ignored)
	if err != nil {
		return nil, err
	}

	s := &Syncer{
		basedir: abs,
		ignore:  re,
	}

	return s, nil
}

func (s Syncer) listFiles() (map[string][]byte, error) {
	list := map[string][]byte{}

	err := filepath.Walk(s.basedir, func(path string, info os.FileInfo, err error) error {
		path = strings.TrimPrefix(path, s.basedir)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimPrefix(path, "\\")

		if info.IsDir() || s.isIgnored(path) {
			return nil
		}

		list[path] = nil
		return nil
	})
	return list, err
}

func (s Syncer) isIgnored(path string) bool {
	return s.ignore.MatchString(path)
}

func (s Syncer) deleteFiles(del map[string][]byte) error {
	for file := range del {
		if s.isIgnored(file) {
			continue
		}
		path := filepath.Join(s.basedir, file)
		err := os.Remove(path)
		if err != nil {
			return err
		}
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
	if del {
		haveFiles, err := s.listFiles()
		if err != nil {
			return fmt.Errorf("failed while listing files in '%s', error is: %s", s.basedir, err.Error())
		}

		_, obsolete, _ := findCommonFiles(haveFiles, files)

		if len(obsolete) > 0 {
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

func findCommonFiles(a, b map[string][]byte) (common, onlyA, onlyB map[string][]byte) {
	common = map[string][]byte{}
	onlyA = map[string][]byte{}
	for f, data := range a {
		if _, ok := b[f]; !ok {
			onlyA[f] = data
		} else {
			common[f] = data
		}
	}

	onlyB = map[string][]byte{}
	for f, data := range b {
		if _, ok := a[f]; !ok {
			onlyB[f] = data
		}
	}

	return
}

func (s Syncer) writeFile(name string, data []byte) error {
	if s.isIgnored(name) {
		return nil
	}

	path := filepath.Join(s.basedir, name)
	//log.Info(fmt.Sprintf("Writing file '%s'...", path))

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

	fileLen, err := file.Write(data)
	if err != nil {
		return err
	}

	err = os.Truncate(path, int64(fileLen))
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
