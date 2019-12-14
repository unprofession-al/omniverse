package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config represents the data structure of the config file. It also
// holds a few helper methods.
type Config struct {
	Singularity singularityConfig `yaml:"singularity" json:"singularity"`
	Alterverses alterverseConfig  `yaml:"alterverses" json:"alterverses"`
}

// NewConfig read the given yaml file and returns a config struct. It returns also
// a slice of valiation errors (which should cause the program to quit) as well as
// the file operation or parsing error if applicable.
func NewConfig(path string) (c *Config, valErrs []error, err error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("error while reading config file %s: %s", path, err)
		return
	}

	c = &Config{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		err = fmt.Errorf("error while unmarshalling config file %s: %s", path, err)
		return
	}

	valErrs = c.Validate()
	return
}

// Validate runs all various Validation tests and returns a slice of all errors
// found.
func (c Config) Validate() []error {
	checker := NewChecker()
	errs := []error{}
	errs = append(errs, checker.ValidateExpressionTemplate(c.Singularity, c.Alterverses)...)
	return errs
}
