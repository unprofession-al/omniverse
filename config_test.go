package main

import (
	"strings"
	"testing"
)

func TestExpressionValidation(t *testing.T) {
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

	for _, test := range tests {
		c := config{
			Singularity: singularity{
				Expression:         test.expression,
				ExpressionTemplate: test.expressionTemplate,
			},
			Alterverses: map[string]map[string]string{
				"test": map[string]string{
					test.definitionKey: "does_not_matter",
				},
			},
		}

		errs := c.ValidateExpressionTemplate()
		hasErr := len(errs) > 0
		if test.expectErr && !hasErr {
			t.Errorf("An error was expected for expression '%s' and template '%s' with key '%s', but validation was ok", test.expression, test.expressionTemplate, test.definitionKey)
		} else if !test.expectErr && hasErr {
			errStrings := []string{}
			for _, err := range errs {
				errStrings = append(errStrings, err.Error())
			}
			t.Errorf("No error expected for expression '%s' and template '%s' with key '%s', but validation returned errors: %v", test.expression, test.expressionTemplate, test.definitionKey, strings.Join(errStrings, " AND "))
		}
	}
}
