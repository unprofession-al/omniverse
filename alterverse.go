package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const alterverseFile = ".alterverse.yml"

type Manifest map[string]string

type Alterverse struct {
	Manifest Manifest `json:"manifest" yaml:"manifest"`
	location string   `json:"location" yaml:"location"`
}

func NewAlterverse(location string) (*Alterverse, []error) {
	a := &Alterverse{location: location}

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

	errs := a.HasValueDublicates()
	return a, errs
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
