package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type config struct {
	Singularity singularity      `yaml:"singularity" json:"singularity"`
	Alterverses alterverseConfig `yaml:"alterverses" json:"alterverses"`
}

func NewConfig(path string) (*config, error) {
	c := &config{}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		errOut := fmt.Errorf("Error while reading config file %s: %s\n", path, err)
		return c, errOut
	}

	err = yaml.Unmarshal(yamlFile, c)

	return c, nil
}
