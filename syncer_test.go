package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

const testdata = "testdata"

func TestCommonFiles(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		a, b, common, onlyA, onlyB []string
	}{
		"Overlapping": {
			a:      []string{"a", "b", "c", "d"},
			b:      []string{"b", "c", "d", "e"},
			common: []string{"b", "c", "d"},
			onlyA:  []string{"a"},
			onlyB:  []string{"e"},
		},
		"OneHasMore": {
			a:      []string{"a", "b", "c", "d"},
			b:      []string{"a", "b", "c", "d", "e"},
			common: []string{"a", "b", "c", "d"},
			onlyA:  []string{},
			onlyB:  []string{"e"},
		},
		"Empty": {
			a:      []string{"a", "b", "c", "d"},
			b:      []string{},
			common: []string{},
			onlyA:  []string{"a", "b", "c", "d"},
			onlyB:  []string{},
		},
		"NoCommon": {
			a:      []string{"a", "b"},
			b:      []string{"c", "d"},
			common: []string{},
			onlyA:  []string{"a", "b"},
			onlyB:  []string{"c", "d"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
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

func TestListFile(t *testing.T) {
	t.Parallel()
	expected := map[string][]byte{
		"FileA.txt": nil,
	}

	basepath := filepath.Join(testdata, "alterverse_ok")
	syncer, err := NewSyncer(basepath, defaultIgnore)
	if err != nil {
		t.Errorf("syncer for '%s' could not be created, error was: %s", basepath, err.Error())
	}

	list, err := syncer.listFiles()
	if err != nil {
		t.Errorf("syncer for '%s' could not list files, error was: %s", basepath, err.Error())
	}

	if !checkSameFields(list, expected) {
		t.Errorf("files in '%s' are not as expected: is %v, expected %v", basepath, list, expected)
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()
	expected := map[string][]byte{
		"FileA.txt": []byte("This is testdata containing bar1 and foo1"),
	}

	basepath := filepath.Join(testdata, "alterverse_ok")
	syncer, err := NewSyncer(basepath, defaultIgnore)
	if err != nil {
		t.Errorf("syncer for '%s' could not be created, error was: %s", basepath, err.Error())
	}

	files, err := syncer.ReadFiles()
	if err != nil {
		t.Errorf("syncer for '%s' could not read files, error was: %s", basepath, err.Error())
	}

	if !checkSameFields(files, expected) {
		t.Errorf("files in '%s' are not as expected: is %v, expected %v", basepath, files, expected)
	}

	for name, data := range expected {
		fsdata, ok := files[name]
		if !ok {
			t.Errorf("file '%s' was expected to be present but was not", name)
			continue
		}
		if bytes.Equal(fsdata, data) {
			t.Errorf("content in file '%s' is not as expected", name)
		}
	}
}

func asMap(in []string) map[string][]byte {
	out := map[string][]byte{}
	for _, k := range in {
		out[k] = nil
	}
	return out
}

func checkSameFields(x, y map[string][]byte) bool {
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
