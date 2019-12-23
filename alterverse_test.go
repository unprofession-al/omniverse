package main

import (
	"fmt"
	"testing"
)

func TestValueDublicates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		manifest          map[string]string
		dublicateExpected bool
	}{
		{
			manifest: map[string]string{
				"foo":  "bar",
				"test": "bla",
			},
			dublicateExpected: false,
		},
		{
			manifest: map[string]string{
				"foo":    "bar",
				"test":   "bla",
				"foobar": "blaa",
			},
			dublicateExpected: false,
		},
		{
			manifest: map[string]string{
				"foo":    "bar",
				"test":   "bla",
				"foobar": "blaa",
				"bar":    "bla",
			},
			dublicateExpected: true,
		},
	}

	for nr, test := range tests {
		t.Run(fmt.Sprintf("%d", nr), func(t *testing.T) {
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
