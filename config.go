package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"text/template"

	"gopkg.in/yaml.v2"
)

// Config represents the data structure of the config file. It also
// holds a few helper methods.
type Config struct {
	Singularity singularity      `yaml:"singularity" json:"singularity"`
	Alterverses alterverseConfig `yaml:"alterverses" json:"alterverses"`
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
	errs := []error{}
	errs = append(errs, c.ValidateExpressionTemplate()...)
	return errs
}

// ValidateExpressionTemplate tests the 'expression' and the 'expression_template'
// given in the sigularity on order to ensure that the conversion from a alterverse
// to the singularity and vice versa delievers consistent results
func (c Config) ValidateExpressionTemplate() []error {
	out := []error{}
	tmpl, err := template.New("expression").Parse(c.Singularity.ExpressionTemplate)
	if err != nil {
		out = append(out, err)
		return out
	}

	re, err := regexp.Compile(c.Singularity.Expression)
	if err != nil {
		out = append(out, err)
		return out
	}

	for name, alterverse := range c.Alterverses {
		for k := range alterverse {
			var expression bytes.Buffer
			err = tmpl.Execute(&expression, k)
			if err != nil {
				err = fmt.Errorf("error occurred while executing expression template for key '%s' in alterverse '%s': %s", k, name, err.Error())
				out = append(out, err)
				continue
			}

			sm := re.FindAllSubmatch(expression.Bytes(), -1)
			if len(sm) > 1 {
				err := fmt.Errorf("expression '%s' matches more than one time with generated place holder '%s' for key '%s' in alterverse '%s'", c.Singularity.Expression, expression.String(), k, name)
				out = append(out, err)
				continue
			}
			if len(sm) < 1 {
				err := fmt.Errorf("expression '%s' does not match with generated place holder '%s' for key '%s' in alterverse '%s'", c.Singularity.Expression, expression.String(), k, name)
				out = append(out, err)
				continue
			}
			if len(sm[0]) < 2 {
				err := fmt.Errorf("expression '%s' seems not to contain the required match group for key '%s' in alterverse '%s'", c.Singularity.Expression, k, name)
				out = append(out, err)
				continue
			}
			if len(sm[0]) > 2 {
				err := fmt.Errorf("expression '%s' seems to have more than one match group for key '%s' in alterverse '%s'", c.Singularity.Expression, k, name)
				out = append(out, err)
				continue
			}
			if string(sm[0][0]) != expression.String() {
				err := fmt.Errorf("expression '%s' seems to match only a substring of generated place holder '%s' key '%s' in alterverse '%s'", c.Singularity.Expression, expression.String(), k, name)
				out = append(out, err)
				continue
			}
			if string(sm[0][1]) != k {
				err := fmt.Errorf("the match group of expression '%s' deos not match with key '%s' in alterverse '%s'", c.Singularity.Expression, k, name)
				out = append(out, err)
				continue
			}
		}
	}

	return out
}
