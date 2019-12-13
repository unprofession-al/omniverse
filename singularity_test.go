package main

import (
	"testing"
)

func TestSingularityFindKeysInFile(t *testing.T) {
	tests := []struct {
		file         []byte
		keysExpected []string
		expression   string
	}{
		{
			file:         []byte(`<<bla>>`),
			keysExpected: []string{"bla"},
			expression:   `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`,
		},
		{
			file:         []byte(`<< bla>>`),
			keysExpected: []string{"bla"},
			expression:   `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`,
		},
		{
			file:         []byte(`<< bla>> foo <<bar>>`),
			keysExpected: []string{"bla", "bar"},
			expression:   `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`,
		},
		{
			file: []byte(`<< bla>> foo <<bar>>,
<<bar>>`),
			keysExpected: []string{"bla", "bar"},
			expression:   `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`,
		},
		{
			file: []byte(`<< bla>> foo <<bar>>,
<<foo>> << bla >>`),
			keysExpected: []string{"bla", "bar", "foo"},
			expression:   `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`,
		},
		{
			file: []byte(`<< bla>> foo <<ba
r>>, <<foo>> << bla >>`),
			keysExpected: []string{"bla", "foo"},
			expression:   `\<\<\W*([a-zA-Z0-9_]+)\W*\>\>`,
		},
	}

	for _, test := range tests {
		s := singularity{
			Expression: test.expression,
		}

		keysFound, err := s.findKeysInFile(test.file)
		if err != nil {
			t.Errorf("error occured while inspecting data: %s", err.Error())
		}

		if len(keysFound) != len(test.keysExpected) {
			t.Errorf("expected keys are %v, found keys are %v", test.keysExpected, keysFound)
		}

		allFound := true
		for _, ke := range test.keysExpected {
			found := false
			for kf := range keysFound {
				if kf == ke {
					found = true
					continue
				}
			}

			if !found {
				allFound = false
				break
			}
		}

		if !allFound {
			t.Errorf("expected keys are %v, found keys are %v", test.keysExpected, keysFound)
		}
	}
}
