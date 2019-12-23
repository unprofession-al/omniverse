package main

import (
	"fmt"
	"testing"
)

func TestCommonFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b, common, onlyA, onlyB []string
	}{
		{
			a:      []string{"a", "b", "c", "d"},
			b:      []string{"b", "c", "d", "e"},
			common: []string{"b", "c", "d"},
			onlyA:  []string{"a"},
			onlyB:  []string{"e"},
		},
		{
			a:      []string{"a", "b", "c", "d"},
			b:      []string{"a", "b", "c", "d", "e"},
			common: []string{"a", "b", "c", "d"},
			onlyA:  []string{},
			onlyB:  []string{"e"},
		},
		{
			a:      []string{"a", "b", "c", "d"},
			b:      []string{},
			common: []string{},
			onlyA:  []string{"a", "b", "c", "d"},
			onlyB:  []string{},
		},
	}

	asMap := func(in []string) map[string][]byte {
		out := map[string][]byte{}
		for _, k := range in {
			out[k] = nil
		}
		return out
	}

	checkSameFields := func(x, y map[string][]byte) bool {
		for k := range x {
			if _, ok := y[k]; !ok {
				return false
			}
		}
		for k := range y {
			if _, ok := x[k]; !ok {
				return false
			}
		}
		return true
	}

	for nr, test := range tests {
		t.Run(fmt.Sprintf("%d", nr), func(t *testing.T) {
			common, onlyA, onlyB := findCommonFiles(asMap(test.a), asMap(test.b))
			if !checkSameFields(common, asMap(test.common)) {
				t.Errorf("common files are not as expected: is %v, expected %v", common, asMap(test.common))
			}
			if !checkSameFields(onlyA, asMap(test.onlyA)) {
				t.Errorf("onlyA files are not as expected: is %v, expected %v", onlyA, asMap(test.onlyA))
			}
			if !checkSameFields(onlyB, asMap(test.onlyB)) {
				t.Errorf("onlyB files are not as expected: is %v, expected %v", onlyB, asMap(test.onlyB))
			}
		})
	}
}
