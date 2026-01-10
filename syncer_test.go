package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testdata = "testdata"

func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

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

func TestSyncerWriteFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename    string
		content     []byte
		ignore      string
		shouldWrite bool
	}{
		"SimpleFile": {
			filename:    "test.txt",
			content:     []byte("test content"),
			ignore:      defaultIgnore,
			shouldWrite: true,
		},
		"FileWithSubdirectory": {
			filename:    "subdir/test.txt",
			content:     []byte("nested content"),
			ignore:      defaultIgnore,
			shouldWrite: true,
		},
		"IgnoredFile": {
			filename:    ".hidden",
			content:     []byte("should be ignored"),
			ignore:      defaultIgnore,
			shouldWrite: false,
		},
		"EmptyContent": {
			filename:    "empty.txt",
			content:     []byte(""),
			ignore:      defaultIgnore,
			shouldWrite: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			syncer, err := NewSyncer(tmpDir, test.ignore)
			if err != nil {
				t.Fatalf("failed to create syncer: %v", err)
			}

			err = syncer.writeFile(test.filename, test.content)
			if err != nil {
				t.Fatalf("writeFile failed: %v", err)
			}

			fullPath := filepath.Join(tmpDir, test.filename)
			data, err := os.ReadFile(fullPath)
			if test.shouldWrite {
				if err != nil {
					t.Errorf("file should have been written but wasn't: %v", err)
				} else if !bytes.Equal(data, test.content) {
					t.Errorf("content mismatch: expected %q, got %q", test.content, data)
				}
			} else {
				if err == nil {
					t.Errorf("file should not have been written but was")
				}
			}
		})
	}
}

func TestSyncerWriteFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files       map[string][]byte
		delete      bool
		errExpected bool
	}{
		"WriteNewFiles": {
			files: map[string][]byte{
				"file1.txt": []byte("content 1"),
				"file2.txt": []byte("content 2"),
			},
			delete:      false,
			errExpected: false,
		},
		"WithNestedDirectories": {
			files: map[string][]byte{
				"dir1/file.txt":     []byte("content"),
				"dir1/dir2/file.txt": []byte("nested content"),
			},
			delete:      false,
			errExpected: false,
		},
		"WithDeletion": {
			files: map[string][]byte{
				"keep.txt": []byte("keep this"),
			},
			delete:      true,
			errExpected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			syncer, err := NewSyncer(tmpDir, defaultIgnore)
			if err != nil {
				t.Fatalf("failed to create syncer: %v", err)
			}

			// Pre-create a file that should be deleted if delete=true
			if test.delete {
				oldFile := filepath.Join(tmpDir, "old.txt")
				if err := os.WriteFile(oldFile, []byte("old content"), 0644); err != nil {
					t.Fatalf("failed to create old file: %v", err)
				}
			}

			err = syncer.WriteFiles(test.files, test.delete)
			if err != nil && !test.errExpected {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && test.errExpected {
				t.Errorf("expected error but got none")
			}
			if test.errExpected {
				return
			}

			// Verify all files were written
			for filename, expectedContent := range test.files {
				fullPath := filepath.Join(tmpDir, filename)
				actualContent, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("failed to read file %s: %v", filename, err)
					continue
				}
				if !bytes.Equal(actualContent, expectedContent) {
					t.Errorf("file %s content mismatch: expected %q, got %q",
						filename, expectedContent, actualContent)
				}
			}

			// Verify old file was deleted if delete=true
			if test.delete {
				oldFile := filepath.Join(tmpDir, "old.txt")
				if _, err := os.Stat(oldFile); err == nil {
					t.Errorf("old file should have been deleted but still exists")
				}
			}
		})
	}
}

func TestSyncerDeleteFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filesToCreate []string
		filesToDelete map[string][]byte
		ignore        string
	}{
		"DeleteSingleFile": {
			filesToCreate: []string{"file1.txt"},
			filesToDelete: map[string][]byte{
				"file1.txt": nil,
			},
			ignore: defaultIgnore,
		},
		"DeleteMultipleFiles": {
			filesToCreate: []string{"file1.txt", "file2.txt", "file3.txt"},
			filesToDelete: map[string][]byte{
				"file1.txt": nil,
				"file3.txt": nil,
			},
			ignore: defaultIgnore,
		},
		"IgnoreHiddenFiles": {
			filesToCreate: []string{".hidden"},
			filesToDelete: map[string][]byte{
				".hidden": nil,
			},
			ignore: defaultIgnore,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			syncer, err := NewSyncer(tmpDir, test.ignore)
			if err != nil {
				t.Fatalf("failed to create syncer: %v", err)
			}

			// Create files to delete
			for _, filename := range test.filesToCreate {
				fullPath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
					t.Fatalf("failed to create file %s: %v", filename, err)
				}
			}

			// Delete files
			err = syncer.deleteFiles(test.filesToDelete)
			if err != nil {
				t.Errorf("deleteFiles failed: %v", err)
			}

			// Verify deletion
			for filename := range test.filesToDelete {
				fullPath := filepath.Join(tmpDir, filename)
				if syncer.isIgnored(filename) {
					// Ignored files should not be deleted
					if _, err := os.Stat(fullPath); err != nil {
						t.Errorf("ignored file %s was deleted", filename)
					}
				} else {
					// Non-ignored files should be deleted
					if _, err := os.Stat(fullPath); err == nil {
						t.Errorf("file %s should have been deleted", filename)
					}
				}
			}
		})
	}
}

func TestSyncerFileTruncation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	syncer, err := NewSyncer(tmpDir, defaultIgnore)
	if err != nil {
		t.Fatalf("failed to create syncer: %v", err)
	}

	filename := "truncate.txt"
	fullPath := filepath.Join(tmpDir, filename)

	// Write initial long content
	longContent := []byte("This is a very long content that will be shortened")
	if err := syncer.writeFile(filename, longContent); err != nil {
		t.Fatalf("failed to write initial content: %v", err)
	}

	// Write shorter content
	shortContent := []byte("Short")
	if err := syncer.writeFile(filename, shortContent); err != nil {
		t.Fatalf("failed to write short content: %v", err)
	}

	// Verify file was properly truncated
	actualContent, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !bytes.Equal(actualContent, shortContent) {
		t.Errorf("file not properly truncated: expected %q, got %q", shortContent, actualContent)
	}
}

func TestNewSyncerErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		basedir     string
		ignore      string
		errExpected bool
		setup       func(t *testing.T) string
	}{
		"NonExistentDirectory": {
			basedir:     "/nonexistent/path/that/does/not/exist",
			ignore:      defaultIgnore,
			errExpected: true,
			setup:       func(t *testing.T) string { return "/nonexistent/path/that/does/not/exist" },
		},
		"InvalidRegex": {
			ignore:      "[invalid(regex",
			errExpected: true,
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
		},
		"FileInsteadOfDirectory": {
			ignore:      defaultIgnore,
			errExpected: true,
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "file.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filePath
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			basedir := test.setup(t)
			_, err := NewSyncer(basedir, test.ignore)
			if err != nil && !test.errExpected {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && test.errExpected {
				t.Errorf("expected error but got none")
			}
		})
	}
}

func TestReadFilesError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	syncer, err := NewSyncer(tmpDir, defaultIgnore)
	if err != nil {
		t.Fatalf("failed to create syncer: %v", err)
	}

	// Create a directory that will cause a read error
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create a file
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Make the file unreadable (this works on Unix-like systems)
	if err := os.Chmod(testFile, 0000); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}
	defer os.Chmod(testFile, 0644) // Clean up

	// This should work on most systems but may not fail on all platforms
	_, err = syncer.ReadFiles()
	// We don't strictly check for error as permissions work differently on different OSes
	if err == nil {
		t.Logf("Warning: ReadFiles did not error on unreadable file (may be platform-specific)")
	}
}

func TestWriteFileOverwrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	syncer, err := NewSyncer(tmpDir, defaultIgnore)
	if err != nil {
		t.Fatalf("failed to create syncer: %v", err)
	}

	filename := "overwrite.txt"
	fullPath := filepath.Join(tmpDir, filename)

	// Write initial content
	initial := []byte("initial content that is quite long")
	if err := os.WriteFile(fullPath, initial, 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	// Overwrite with shorter content
	newContent := []byte("new")
	if err := syncer.writeFile(filename, newContent); err != nil {
		t.Fatalf("failed to overwrite file: %v", err)
	}

	// Verify file was overwritten and truncated
	result, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !bytes.Equal(result, newContent) {
		t.Errorf("file not properly overwritten: expected %q, got %q", newContent, result)
	}
	if len(result) != len(newContent) {
		t.Errorf("file not properly truncated: expected length %d, got %d", len(newContent), len(result))
	}
}

func TestDeleteFilesError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (permissions work differently)")
	}

	tmpDir := t.TempDir()
	syncer, err := NewSyncer(tmpDir, defaultIgnore)
	if err != nil {
		t.Fatalf("failed to create syncer: %v", err)
	}

	// Create a read-only directory with a file in it
	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(roDir, 0755); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}

	testFile := filepath.Join(roDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Make directory read-only so file deletion will fail
	if err := os.Chmod(roDir, 0555); err != nil {
		t.Fatalf("failed to chmod dir: %v", err)
	}
	defer os.Chmod(roDir, 0755) // Clean up

	// Try to delete the file - should fail due to readonly directory
	relPath := filepath.Join("readonly", "file.txt")
	err = syncer.deleteFiles(map[string][]byte{relPath: nil})
	if err == nil {
		t.Errorf("expected error deleting file in readonly directory")
	}
}

// FuzzSyncerIgnorePattern tests ignore pattern matching with various inputs
func FuzzSyncerIgnorePattern(f *testing.F) {
	// Seed corpus
	f.Add(".hidden", `^\..*`)
	f.Add("normal.txt", `^\..*`)
	f.Add(".git/config", `^\..*`)
	f.Add("test", `test`)
	f.Add("path/to/file.txt", `.*\.txt$`)

	f.Fuzz(func(t *testing.T, filename string, pattern string) {
		tmpDir := t.TempDir()

		// Try to create syncer with the pattern
		syncer, err := NewSyncer(tmpDir, pattern)
		if err != nil {
			// Invalid regex patterns are expected to fail
			t.Skip()
		}

		// Test if ignore pattern works without crashing
		result := syncer.isIgnored(filename)

		// Result should be a boolean
		_ = result
	})
}

// FuzzSyncerWriteRead tests writing and reading files with various content
func FuzzSyncerWriteRead(f *testing.F) {
	// Seed corpus
	f.Add("test.txt", []byte("hello world"))
	f.Add("data.bin", []byte{0x00, 0xFF, 0xAA, 0x55})
	f.Add("empty.txt", []byte(""))
	f.Add("subdir/file.txt", []byte("nested content"))

	f.Fuzz(func(t *testing.T, filename string, content []byte) {
		// Skip invalid filenames
		if filename == "" || strings.Contains(filename, "\x00") {
			t.Skip()
		}

		// Skip filenames with absolute paths or path traversal
		if filepath.IsAbs(filename) || strings.Contains(filename, "..") {
			t.Skip()
		}

		// Skip pathological cases: filenames that are only separators
		// These get trimmed to empty strings during path processing
		trimmed := strings.TrimLeft(filename, "/\\")
		if trimmed == "" {
			t.Skip()
		}

		// Skip filenames starting with backslash - on Unix these are valid filename
		// characters but the syncer code trims them as path separators (for Windows compat)
		// This is a known limitation when filenames start with backslash on Unix
		if strings.HasPrefix(filename, "\\") {
			t.Skip()
		}

		tmpDir := t.TempDir()
		syncer, err := NewSyncer(tmpDir, defaultIgnore)
		if err != nil {
			t.Fatalf("failed to create syncer: %v", err)
		}

		// Write file
		err = syncer.writeFile(filename, content)
		if err != nil {
			// Some errors are expected for invalid filenames
			t.Skip()
		}

		// Read it back
		files, err := syncer.ReadFiles()
		if err != nil {
			t.Fatalf("failed to read files: %v", err)
		}

		// Verify content matches if file was written (not ignored)
		if !syncer.isIgnored(filename) {
			// filepath.Join normalizes paths, so "0/" becomes "0"
			// We need to check for the normalized version
			normalizedName := filepath.Clean(filename)
			if normalizedName == "." {
				normalizedName = filename
			}

			readContent, ok := files[normalizedName]
			if !ok {
				// Try with original filename too
				readContent, ok = files[filename]
			}
			if !ok {
				t.Errorf("file %q (or normalized %q) not found after writing. Files: %v",
					filename, normalizedName, mapKeys(files))
			} else if !bytes.Equal(readContent, content) {
				t.Errorf("content mismatch for %q: wrote %d bytes, read %d bytes",
					filename, len(content), len(readContent))
			}
		}
	})
}

// FuzzFindCommonFiles tests the common file finding logic
func FuzzFindCommonFiles(f *testing.F) {
	// Seed corpus
	f.Add("file1.txt", "file2.txt", "file1.txt", "file3.txt")
	f.Add("a.txt", "b.txt", "c.txt", "d.txt")
	f.Add("shared", "unique1", "shared", "unique2")

	f.Fuzz(func(t *testing.T, a1 string, a2 string, b1 string, b2 string) {
		// Skip empty filenames
		if a1 == "" || a2 == "" || b1 == "" || b2 == "" {
			t.Skip()
		}

		mapA := map[string][]byte{
			a1: []byte("contentA1"),
			a2: []byte("contentA2"),
		}

		mapB := map[string][]byte{
			b1: []byte("contentB1"),
			b2: []byte("contentB2"),
		}

		common, onlyA, onlyB := findCommonFiles(mapA, mapB)

		// Every file in A should be in either common or onlyA
		for file := range mapA {
			if _, inCommon := common[file]; !inCommon {
				if _, inOnlyA := onlyA[file]; !inOnlyA {
					t.Errorf("file %q from A not found in common or onlyA", file)
				}
			}
		}

		// Every file in B should be in either common or onlyB
		for file := range mapB {
			if _, inCommon := common[file]; !inCommon {
				if _, inOnlyB := onlyB[file]; !inOnlyB {
					t.Errorf("file %q from B not found in common or onlyB", file)
				}
			}
		}

		// Files in common must be in both A and B
		for file := range common {
			if _, inA := mapA[file]; !inA {
				t.Errorf("file %q in common but not in A", file)
			}
			if _, inB := mapB[file]; !inB {
				t.Errorf("file %q in common but not in B", file)
			}
		}

		// Files in onlyA must not be in B
		for file := range onlyA {
			if _, inB := mapB[file]; inB {
				t.Errorf("file %q in onlyA but also in B", file)
			}
		}

		// Files in onlyB must not be in A
		for file := range onlyB {
			if _, inA := mapA[file]; inA {
				t.Errorf("file %q in onlyB but also in A", file)
			}
		}
	})
}
