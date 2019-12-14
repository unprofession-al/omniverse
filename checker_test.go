package main

import (
	"strings"
	"testing"
)

func TestCheckerExpressionTemplateValidation(t *testing.T) {
	tests := []struct {
		expression         string
		expressionTemplate string
		definitionKey      string
		expectErr          bool
	}{
		// everything as it should
		{`\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`, "<< {{.}} >>", "test", false},

		// the template is broken
		{`\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`, "<< {{.} >>", "test", true},

		// the expression is broken
		{`\<\<\W*([a-zA-Z0-9_]+\W*\>\>`, "<< {{.}} >>", "test", true},

		// the rendered template results in multiple matches
		{`\<\<\W*([a-zA-Z0-9_])+\W*\>\>`, "<< {{.}} >><< {{.}} >>", "test", true},

		// the key is dublicated in the rendered template
		{`\<\<\W*([a-zA-Z0-9_])+\W*\>\>`, "<< {{.}}{{.}} >>", "test", true},

		// the template contains too characters in addition to the string
		// that matches the expression
		{`\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`, "<< {{.}} >>__", "test", true},

		// the expression does not contain the required match group
		{`\<\<\W*[a-zA-Z0-9_]+\W*\>\>`, "<< {{.}} >>", "test", true},

		// the expression does not contains more than one match group
		{`\<\<\W*([a-zA-Z0-9_]+)(\[a-zA-Z0-9_]+)\W*\>\>`, "<< {{.}} >>", "test", true},

		// the match group does not match the key
		{`\<\<\W*([A-Z0-9_]+)\W*\>\>`, "<< {{.}} >>", "test", true},
	}

	c := NewChecker()

	for _, test := range tests {
		s := singularityConfig{
			Expression:         test.expression,
			ExpressionTemplate: test.expressionTemplate,
		}
		a := alterverseConfig{
			"test": {
				test.definitionKey: "does_not_matter",
			},
		}

		errs := c.ValidateExpressionTemplate(s, a)
		hasErr := len(errs) > 0
		if test.expectErr && !hasErr {
			t.Errorf("an error was expected for expression '%s' and template '%s' with key '%s', but validation was ok", test.expression, test.expressionTemplate, test.definitionKey)
		} else if !test.expectErr && hasErr {
			errStrings := []string{}
			for _, err := range errs {
				errStrings = append(errStrings, err.Error())
			}
			t.Errorf("no error expected for expression '%s' and template '%s' with key '%s', but validation returned errors: %v", test.expression, test.expressionTemplate, test.definitionKey, strings.Join(errStrings, " AND "))
		}
	}
}

func TestCheckerValidateDefinitionIfDefinitionsAreObsolete(t *testing.T) {
	tests := []struct {
		definitions map[string]string
		expectErr   bool
	}{
		{
			definitions: map[string]string{
				"foo": "does_not_matter",
				"bar": "does_not_matter",
				"bla": "does_not_matter",
			},
			expectErr: false,
		},
		{
			definitions: map[string]string{
				"foo": "does_not_matter",
				"bar": "does_not_matter",
			},
			expectErr: false,
		},
		{
			definitions: map[string]string{
				"foo": "does_not_matter",
				"bar": "does_not_matter",
				"bla": "does_not_matter",
				"NOT": "does_not_matter",
			},
			expectErr: true,
		},
	}

	files := map[string][]byte{
		"foobar": []byte(`foo <<foo>> bar <<bar>>`),
		"bla":    []byte(`bla <<bla>>`),
	}
	sc := singularityConfig{Expression: `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`}
	s, err := NewSingularity(sc, files)
	if err != nil {
		t.Errorf("could not read singularity: %s", err.Error())
	}

	c := NewChecker()
	for _, test := range tests {
		errs := c.ValidateDefinitionIfDefinitionsAreObsolete(test.definitions, *s)
		hasErr := len(errs) > 0
		if test.expectErr && !hasErr {
			t.Errorf("errors were expected for definitions '%v', but validation was ok", test.definitions)
		} else if !test.expectErr && hasErr {
			t.Errorf("no errors were expected for definitions '%v', but errors occurred: %v", test.definitions, errs)
		}
	}
}

func TestCheckerValidateSingularityIfKeysDefined(t *testing.T) {
	tests := []struct {
		definitions map[string]string
		expectErr   bool
	}{
		{
			definitions: map[string]string{
				"foo": "does_not_matter",
				"bar": "does_not_matter",
				"bla": "does_not_matter",
			},
			expectErr: false,
		},
		{
			definitions: map[string]string{
				"foo": "does_not_matter",
				"bar": "does_not_matter",
			},
			expectErr: true,
		},
		{
			definitions: map[string]string{
				"foo": "does_not_matter",
				"bar": "does_not_matter",
				"bla": "does_not_matter",
				"NOT": "does_not_matter",
			},
			expectErr: false,
		},
	}

	files := map[string][]byte{
		"foobar": []byte(`foo <<foo>> bar <<bar>>`),
		"bla":    []byte(`bla <<bla>>`),
	}
	sc := singularityConfig{Expression: `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`}
	s, err := NewSingularity(sc, files)
	if err != nil {
		t.Errorf("could not read singularity: %s", err.Error())
	}

	c := NewChecker()
	for _, test := range tests {
		errs := c.ValidateSingularityIfKeysDefined(*s, test.definitions)
		hasErr := len(errs) > 0
		if test.expectErr && !hasErr {
			t.Errorf("errors were expected for definitions '%v', but validation was ok", test.definitions)
		} else if !test.expectErr && hasErr {
			t.Errorf("no errors were expected for definitions '%v', but errors occurred: %v", test.definitions, errs)
		}
	}
}
