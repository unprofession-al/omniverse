package main

import (
	"reflect"
	"testing"
)

func TestDiffFiles(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		a        map[string][]byte
		b        map[string][]byte
		diffs    map[string]bool
		obsolete map[string][]byte
		created  map[string][]byte
	}{
		"NoFiles": {
			a:        map[string][]byte{},
			b:        map[string][]byte{},
			diffs:    map[string]bool{},
			obsolete: map[string][]byte{},
			created:  map[string][]byte{},
		},
		"Similar": {
			a: map[string][]byte{
				"a": []byte(`a`),
				"b": []byte(`b`),
			},
			b: map[string][]byte{
				"a": []byte(`a`),
				"b": []byte(`b`),
			},
			diffs: map[string]bool{
				"a": false,
				"b": false,
			},
			obsolete: map[string][]byte{},
			created:  map[string][]byte{},
		},
		"HasDiff": {
			a: map[string][]byte{
				"a": []byte(`aa`),
				"b": []byte(`b`),
			},
			b: map[string][]byte{
				"a": []byte(`a`),
				"b": []byte(`b`),
			},
			diffs: map[string]bool{
				"a": true,
				"b": false,
			},
			obsolete: map[string][]byte{},
			created:  map[string][]byte{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diffs, obsolete, created := DiffFiles(test.a, test.b)
			for file, diff := range diffs {
				shouldHaveDiff, ok := test.diffs[file]
				if !ok {
					t.Errorf("test case has nothing to compare, please fix")
				}

				hasDiff := diff != ""
				if hasDiff != shouldHaveDiff {
					t.Errorf("diffs is not as expected")
				}
			}
			if !reflect.DeepEqual(obsolete, test.obsolete) {
				t.Errorf("obsolete is not as expected")
			}
			if !reflect.DeepEqual(created, test.created) {
				t.Errorf("created is not as expected")
			}
		})
	}

}
