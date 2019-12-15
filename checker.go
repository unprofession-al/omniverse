package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// Checker holds all methods to find potentiol error sources. It stores the errors found.
type Checker struct {
	Errs map[string][]error
}

func NewChecker() Checker {
	return Checker{Errs: map[string][]error{}}
}

// ValidateSingularityIfKeysDefined checks if all keys found in the singularity are defined
// in the alterverse. If that would not be the case those strings could not be substituted
// properly.
func (c *Checker) ValidateSingularityIfKeysDefined(s singularity, definitions map[string]string) []error {
	checkName := "Validate singularity if all required keys are present in alterverse definitions"
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
	c.Errs[checkName] = out
	return out
}

// ValidateDefinitionIfDefinitionsAreObsolete checks of all definitions are required by the singularity
// in order do properly render. If thats not the case nothing bad happens, its just the alterverse
// definition that is more cluttered than necessary.
func (c *Checker) ValidateDefinitionIfDefinitionsAreObsolete(definitions map[string]string, s singularity) []error {
	checkName := "Validate if all definitions in the alterverse are required by singularity"
	out := []error{}
	keys := s.GetKeys()
	for k := range definitions {
		if _, ok := keys[k]; !ok {
			err := fmt.Errorf("definition of key '%s' present but key does not exist in singularity ", k)
			out = append(out, err)
		}
	}
	c.Errs[checkName] = out
	return out
}

// ValidateExpressionTemplate tests the 'expression' and the 'expression_template'
// given in the sigularity on order to ensure that the conversion from a alterverse
// to the singularity and vice versa delievers consistent results
func (c Checker) ValidateExpressionTemplate(s singularityConfig, a alterverseConfig) []error {
	checkName := "Validate Expression Template"
	out := []error{}
	tmpl, err := template.New("expression").Parse(s.ExpressionTemplate)
	if err != nil {
		out = append(out, err)
		c.Errs[checkName] = out
		return out
	}

	re, err := regexp.Compile(s.Expression)
	if err != nil {
		out = append(out, err)
		c.Errs[checkName] = out
		return out
	}

	for name, alterverse := range a {
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
				err := fmt.Errorf("expression '%s' matches more than one time with generated place holder '%s' for key '%s' in alterverse '%s'", s.Expression, expression.String(), k, name)
				out = append(out, err)
				continue
			}
			if len(sm) < 1 {
				err := fmt.Errorf("expression '%s' does not match with generated place holder '%s' for key '%s' in alterverse '%s'", s.Expression, expression.String(), k, name)
				out = append(out, err)
				continue
			}
			if len(sm[0]) < 2 {
				err := fmt.Errorf("expression '%s' seems not to contain the required match group for key '%s' in alterverse '%s'", s.Expression, k, name)
				out = append(out, err)
				continue
			}
			if len(sm[0]) > 2 {
				err := fmt.Errorf("expression '%s' seems to have more than one match group for key '%s' in alterverse '%s'", s.Expression, k, name)
				out = append(out, err)
				continue
			}
			if string(sm[0][0]) != expression.String() {
				err := fmt.Errorf("expression '%s' seems to match only a substring of generated place holder '%s' key '%s' in alterverse '%s'", s.Expression, expression.String(), k, name)
				out = append(out, err)
				continue
			}
			if string(sm[0][1]) != k {
				err := fmt.Errorf("the match group of expression '%s' deos not match with key '%s' in alterverse '%s'", s.Expression, k, name)
				out = append(out, err)
				continue
			}
		}
	}

	c.Errs[checkName] = out
	return out
}

// ValitateEqualDefinitionValues checks some definitions have equal values strings. If this is true it is
// impossible to deduce the singularity properly
func (c Checker) ValidateEqualDefinitonValues(definitions map[string]string) []error {
	checkName := "Validate if Values of Definitions are equal"
	errs := []error{}

	reverse := reverseStringMap(definitions)
	for v, k := range reverse {
		if len(k) > 1 {
			errs = append(errs, fmt.Errorf("the keys '%s' have the same value '%s' which makes it impossible to deduce the singularity properly", strings.Join(k, ", "), v))
		}
	}

	c.Errs[checkName] = errs
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

// ExpressionHasMatches checks if the files passed have matches. This should be checked when deducing the
// singularity. If true this will lead to confusing results.
func (c Checker) ExpressionHasMatches(expression string, files map[string][]byte) []error {
	checkName := "Validate if files contain strings that match the expression"
	errs := []error{}

	re, err := regexp.Compile(expression)
	if err != nil {
		errs = append(errs, err)
		c.Errs[checkName] = errs
		return errs
	}

	for name, file := range files {
		hasMatch := re.Match(file)
		if hasMatch {
			errs = append(errs, fmt.Errorf("file '%s' contains strings that match the expression '%s'", name, expression))
		}
	}

	return errs
}
