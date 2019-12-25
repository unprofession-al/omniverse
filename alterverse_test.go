package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestNewAlterverse(t *testing.T) {
	t.Parallel()
        tests := []struct {
		location    string
		ignore      string
		errExpected bool
	}{
		{location: "alterverse_ok", ignore: defaultIgnore, errExpected: false},
		{location: "alterverse_empty", ignore: defaultIgnore, errExpected: false},
		{location: "alterverse_does_not_exist", ignore: defaultIgnore, errExpected: true},
		{location: "alterverse_manifest_missing", ignore: defaultIgnore, errExpected: true},
		{location: "alterverse_is_file", ignore: defaultIgnore, errExpected: true},
		{location: "alterverse_malformed_manifest", ignore: defaultIgnore, errExpected: true},
	}

	for _, test := range tests {
		t.Run(test.location, func(t *testing.T) {
			_, errs := NewAlterverse(filepath.Join(testdata, test.location), test.ignore)
			hasErrs := len(errs) > 0
			if hasErrs && !test.errExpected {
				t.Errorf("has unexpected errors, errors are: %v", errs)
			} else if !hasErrs && test.errExpected {
				t.Errorf("errors expected but no errors orrured")
			}
		})
	}
}

func TestValueDublicates(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		manifest          map[string]string
		dublicateExpected bool
	}{
		"Basic": {
			manifest: map[string]string{
				"foo":  "bar",
				"test": "bla",
			},
			dublicateExpected: false,
		},
		"WithSubstring": {
			manifest: map[string]string{
				"foo":    "bar",
				"test":   "bla",
				"foobar": "blaa",
			},
			dublicateExpected: false,
		},
		"WithDublicate": {
			manifest: map[string]string{
				"foo":    "bar",
				"test":   "bla",
				"foobar": "blaa",
				"bar":    "bla",
			},
			dublicateExpected: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a := Alterverse{Manifest: test.manifest}
			errs := a.HasValueDublicates()
			hasDublicates := false
			for _, err := range errs {
				if err != nil {
					hasDublicates = true
				}
			}
			if hasDublicates && !test.dublicateExpected {
				t.Errorf("dublicates found but not expected: %v", errs)
			} else if !hasDublicates && test.dublicateExpected {
				t.Errorf("no dublicates found but expected")
			}
		})
	}
}

func ExampleReverseStringMap() {
	m := map[string]string{
		"foo":    "test",
		"bar":    "bla",
		"foobar": "bla",
	}
	r := reverseStringMap(m)
	fmt.Println(r)

	// Output: map[bla:[bar foobar] test:[foo]]
}
