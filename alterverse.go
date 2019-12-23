package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const alterverseFile = ".alterverse.yml"

// Manifest contains a map of identifiers to thir values.
type Manifest map[string]string

// Alterverse contains specific information per alterverse.
type Alterverse struct {
	Manifest Manifest `json:"manifest" yaml:"manifest"`

	location string
	syncer   *Syncer
}

// NewAlterverse takes a path to a dicectory, reads the manifest file,
// performes necessary checks and returnes the alterverse.
func NewAlterverse(location, ignore string) (*Alterverse, []error) {
	a := &Alterverse{location: location}

	li, err := os.Stat(location)
	if err != nil {
		err = fmt.Errorf("error while checking location '%s': %s", location, err)
		return a, []error{err}
	}
	if !li.IsDir() {
		err = fmt.Errorf("location '%s' does not seem to be a directory", location)
		return a, []error{err}
	}

	manifestPath := filepath.Join(location, alterverseFile)
	manifestFile, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		err = fmt.Errorf("error while reading manifest file '%s': %s", manifestPath, err)
		return a, []error{err}
	}
	err = yaml.Unmarshal(manifestFile, a)
	if err != nil {
		err = fmt.Errorf("error while unmarshalling manifest file '%s': %s", manifestPath, err)
		return a, []error{err}
	}

	a.syncer, err = NewSyncer(location, ignore)
	if err != nil {
		return a, []error{err}
	}

	errs := a.HasValueDublicates()
	return a, errs
}

// Files reads all files related to the alterverse and returns them as a map where the keys are
// the relative file names and the values are the bytes.
func (a Alterverse) Files() (map[string][]byte, error) {
	return a.syncer.ReadFiles()
}

// WriteFiles writes the files passed to the base directory of the alterverse. File names must
// be relative to the alterverse. Files that exist on the file system but not in the map passed
// will be deleted.
func (a Alterverse) WriteFiles(files map[string][]byte) error {
	deleteObselete := true
	return a.syncer.WriteFiles(files, deleteObselete)
}

// HasValueDublicates checks some definitions have equal values strings. If this is true it is
// impossible to deduce the singularity properly
func (a Alterverse) HasValueDublicates() []error {
	errs := []error{}

	reverse := reverseStringMap(a.Manifest)
	for v, k := range reverse {
		if len(k) > 1 {
			errs = append(errs, fmt.Errorf("the keys '%s' have the same value '%s'", strings.Join(k, ", "), v))
		}
	}

	return errs
}

// reverseStringMap switches the keys and values of a map. Since values (of the input)
// can be duplicated (different keys have the same value) the values of the map returned
// is a list of all the keys (of the input map) with this particular value.
func reverseStringMap(in map[string]string) map[string][]string {
	out := make(map[string][]string)
	for k, v := range in {
		if existing, ok := out[v]; ok {
			out[v] = append(existing, k)
		} else {
			out[v] = []string{k}
		}
	}
	return out
}
