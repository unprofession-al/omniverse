package main

import (
	"fmt"
	"os"
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

func TestAlterverseFiles(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		location    string
		ignore      string
		expectedLen int
		errExpected bool
	}{
		"BasicRead": {
			location:    "alterverse_ok",
			ignore:      defaultIgnore,
			expectedLen: 1,
			errExpected: false,
		},
		"EmptyAlterverse": {
			location:    "alterverse_empty",
			ignore:      defaultIgnore,
			expectedLen: 0,
			errExpected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a, errs := NewAlterverse(filepath.Join(testdata, test.location), test.ignore)
			if len(errs) > 0 && !test.errExpected {
				t.Fatalf("unexpected errors creating alterverse: %v", errs)
			}
			if len(errs) == 0 && test.errExpected {
				t.Fatalf("expected errors creating alterverse but got none")
			}
			if test.errExpected {
				return
			}

			files, err := a.Files()
			if err != nil {
				t.Errorf("unexpected error reading files: %v", err)
			}
			if len(files) != test.expectedLen {
				t.Errorf("expected %d files, got %d", test.expectedLen, len(files))
			}
		})
	}
}

func TestAlterverseWriteFiles(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a manifest file in the temp directory
	manifestContent := []byte("manifest:\n  key1: value1\n  key2: value2\n")
	manifestPath := filepath.Join(tmpDir, alterverseFile)
	if err := os.WriteFile(manifestPath, manifestContent, 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	a, errs := NewAlterverse(tmpDir, defaultIgnore)
	if len(errs) > 0 {
		t.Fatalf("failed to create alterverse: %v", errs)
	}

	tests := map[string]struct {
		files       map[string][]byte
		errExpected bool
	}{
		"WriteNewFile": {
			files: map[string][]byte{
				"test.txt": []byte("test content"),
			},
			errExpected: false,
		},
		"WriteMultipleFiles": {
			files: map[string][]byte{
				"test1.txt":     []byte("content 1"),
				"test2.txt":     []byte("content 2"),
				"dir/test3.txt": []byte("content 3"),
			},
			errExpected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := a.WriteFiles(test.files)
			if err != nil && !test.errExpected {
				t.Errorf("unexpected error writing files: %v", err)
			}
			if err == nil && test.errExpected {
				t.Errorf("expected error but got none")
			}
			if test.errExpected {
				return
			}

			// Verify files were written correctly
			for filename, expectedContent := range test.files {
				fullPath := filepath.Join(tmpDir, filename)
				actualContent, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("failed to read written file %s: %v", filename, err)
					continue
				}
				if string(actualContent) != string(expectedContent) {
					t.Errorf("file %s content mismatch: expected %q, got %q",
						filename, expectedContent, actualContent)
				}
			}
		})
	}
}

// FuzzValueDuplicates tests the duplicate value detection with various manifests
func FuzzValueDuplicates(f *testing.F) {
	// Seed corpus
	f.Add("key1", "value1", "key2", "value2")
	f.Add("foo", "bar", "baz", "bar")
	f.Add("a", "x", "b", "y")
	f.Add("test", "", "other", "value")

	f.Fuzz(func(t *testing.T, key1 string, val1 string, key2 string, val2 string) {
		// Skip if keys are the same (not a valid manifest)
		if key1 == key2 || key1 == "" || key2 == "" {
			t.Skip()
		}

		manifest := map[string]string{
			key1: val1,
			key2: val2,
		}

		a := Alterverse{Manifest: manifest}
		errs := a.HasValueDublicates()

		// If values are the same (and both non-empty), should have errors
		if val1 == val2 && val1 != "" {
			if len(errs) == 0 {
				t.Errorf("Expected duplicate error for values %q but got none", val1)
			}
		}

		// If values are different, should have no errors
		if val1 != val2 {
			if len(errs) > 0 {
				t.Errorf("Unexpected duplicate errors: %v", errs)
			}
		}
	})
}

// FuzzReverseStringMap tests the reverse string map function
func FuzzReverseStringMap(f *testing.F) {
	// Seed corpus
	f.Add("key1", "value1", "key2", "value2")
	f.Add("a", "x", "b", "x")
	f.Add("foo", "bar", "baz", "qux")

	f.Fuzz(func(t *testing.T, key1 string, val1 string, key2 string, val2 string) {
		if key1 == key2 || key1 == "" || key2 == "" {
			t.Skip()
		}

		input := map[string]string{
			key1: val1,
			key2: val2,
		}

		result := reverseStringMap(input)

		// Verify all values from input appear as keys in result
		for _, v := range input {
			if _, ok := result[v]; !ok {
				t.Errorf("Value %q from input not found in reversed map", v)
			}
		}

		// Verify the mapping is correct
		for val, keys := range result {
			for _, key := range keys {
				if input[key] != val {
					t.Errorf("Reversed map incorrect: key %q should map to %q, got %q", key, val, input[key])
				}
			}
		}

		// Verify result values are sorted
		for _, keys := range result {
			for i := 1; i < len(keys); i++ {
				if keys[i-1] >= keys[i] {
					t.Errorf("Keys not sorted: %v", keys)
				}
			}
		}
	})
}

// FuzzManifestParsing tests manifest creation with various YAML content
func FuzzManifestParsing(f *testing.F) {
	// Seed corpus with valid manifests
	f.Add("key1", "value1")
	f.Add("url", "example.com")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, key string, value string) {
		// Skip empty keys
		if key == "" {
			t.Skip()
		}

		tmpDir := t.TempDir()

		// Create a manifest YAML file
		manifestContent := fmt.Sprintf("manifest:\n  %s: %s\n", key, value)
		manifestPath := filepath.Join(tmpDir, alterverseFile)

		if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
			t.Skip() // Skip if we can't create the file
		}

		// Try to create alterverse
		a, errs := NewAlterverse(tmpDir, defaultIgnore)

		// Verify manifest was parsed correctly if no errors
		if len(errs) == 0 {
			if a.Manifest[key] != value {
				t.Errorf("Manifest parsing incorrect: expected %q for key %q, got %q",
					value, key, a.Manifest[key])
			}
		}
	})
}
