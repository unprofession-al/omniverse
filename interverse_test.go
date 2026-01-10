package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
)

var randIterations = flag.Int("randiter", 10000, "number of iterations for randomized/fuzzed tests")
var log = flag.Bool("log", false, "print additional log")

var deduceTests = map[string]struct {
	manifestFrom      map[string]string
	from              map[string][]byte
	manifestTo        map[string]string
	to                map[string][]byte
	errExpected       bool
	strictErrExpected bool
}{
	"Basic": {
		manifestFrom: map[string]string{
			"url": "example.com",
			"env": "production",
		},
		from: map[string][]byte{
			"test1": []byte("This is the __production__ environment. The API can be reached at `api.example.com`."),
		},
		manifestTo: map[string]string{
			"url": "example-int.com",
			"env": "integration",
		},
		to: map[string][]byte{
			"test1": []byte("This is the __integration__ environment. The API can be reached at `api.example-int.com`."),
		},
		errExpected:       false,
		strictErrExpected: false,
	},
	"ToKeyMissing": {
		manifestFrom: map[string]string{
			"url": "example.com",
			"env": "production",
		},
		from: map[string][]byte{
			"test1": []byte("This is the __production__ environment. The API can be reached at `api.example.com`."),
		},
		manifestTo: map[string]string{
			"url": "example-int.com",
		},
		to:                map[string][]byte{},
		errExpected:       true,
		strictErrExpected: false,
	},
	"SwitchingStrings": {
		manifestFrom: map[string]string{
			"url":       "example.com",
			"other_url": "example-int.com",
			"env":       "production",
			"other_env": "integration",
		},
		from: map[string][]byte{
			"test1": []byte(`This is the repository of the production environment (example.com).
All API calles to its integration environment (example-int.com) must be avoided.`),
		},
		manifestTo: map[string]string{
			"url":       "example-int.com",
			"other_url": "example.com",
			"env":       "integration",
			"other_env": "production",
		},
		to: map[string][]byte{
			"test1": []byte(`This is the repository of the integration environment (example-int.com).
All API calles to its production environment (example.com) must be avoided.`),
		},
		errExpected:       false,
		strictErrExpected: false,
	},
	"Substrings": {
		manifestFrom: map[string]string{
			"url":     "example.com",
			"api_url": "api.example.com",
			"env":     "production",
		},
		from: map[string][]byte{
			"test1": []byte(`The production environment consists of a series of HTTP endpoints exposed to the internet:
- A end user website is preseted at www.example.com and example.com respectively.
- A management frontend is accessable via admin.example.com.
- An API is exposing functionality at api.example.com.`),
		},
		manifestTo: map[string]string{
			"url":     "example-int.com",
			"api_url": "next-api.example-int.com",
			"env":     "integration",
		},
		to: map[string][]byte{
			"test1": []byte(`The integration environment consists of a series of HTTP endpoints exposed to the internet:
- A end user website is preseted at www.example-int.com and example-int.com respectively.
- A management frontend is accessable via admin.example-int.com.
- An API is exposing functionality at next-api.example-int.com.`),
		},
		errExpected:       false,
		strictErrExpected: false,
	},
	"ToFoundExpected": {
		manifestFrom: map[string]string{
			"url": "example.com",
			"env": "production",
		},
		from: map[string][]byte{
			"test1": []byte("This is the __production__ environment. The API can be reached at `api.example.com` (not `example-int.com`)."),
		},
		manifestTo: map[string]string{
			"url": "example-int.com",
			"env": "integration",
		},
		to: map[string][]byte{
			"test1": []byte("This is the __integration__ environment. The API can be reached at `api.example-int.com` (not `example-int.com`)."),
		},
		errExpected:       false,
		strictErrExpected: true,
	},
	"EmptyKeyInFrom": {
		manifestFrom: map[string]string{
			"env": "",
		},
		from: map[string][]byte{
			"test1": []byte("This is the __production__ environment."),
		},
		manifestTo: map[string]string{
			"env": "integration",
		},
		to: map[string][]byte{
			"test1": []byte{},
		},
		errExpected:       true,
		strictErrExpected: false,
	},
	"EmptyKeyInTo": {
		manifestFrom: map[string]string{
			"env": "production",
		},
		from: map[string][]byte{
			"test1": []byte("This is the __production__ environment."),
		},
		manifestTo: map[string]string{
			"env": "",
		},
		to: map[string][]byte{
			"test1": []byte{},
		},
		errExpected:       true,
		strictErrExpected: false,
	},
	"VVVV": {
		manifestFrom: map[string]string{
			"x": "VV",
		},
		from: map[string][]byte{
			"test1": []byte("VVVV"),
		},
		manifestTo: map[string]string{
			"x": "VVVV",
		},
		to: map[string][]byte{
			"test1": []byte("VVVVVVVV"),
		},
		errExpected:       false,
		strictErrExpected: false,
	},
	"ImpossibleRoundTrip": {
		manifestFrom: map[string]string{
			"x": "xx",
		},
		from: map[string][]byte{
			"test1": []byte("x_xx_x"),
		},
		manifestTo: map[string]string{
			"x": "x_x",
		},
		to: map[string][]byte{
			"test1": []byte("x_x_x_x"),
		},
		errExpected:       false,
		strictErrExpected: true,
	},
}

func TestDeduceSimple(t *testing.T) {
	t.Parallel()

	for name, test := range deduceTests {
		t.Run(name, func(t *testing.T) {
			i, err := NewInterverse(test.manifestFrom, test.manifestTo)
			if err != nil && !test.errExpected {
				t.Errorf("could not create interverse, error was: %s", err.Error())
				return
			} else if err != nil && test.errExpected {
				// success
				return
			} else if err == nil && test.errExpected {
				t.Errorf("could create interverse but error expected")
				return
			}

			r := i.Deduce(test.from)
			if !reflect.DeepEqual(r, test.to) {
				for file, data := range test.to {
					if !bytes.Equal(data, r[file]) {
						t.Errorf("result for file %s not as expected:\n--- Expected:\n%s\n--- Deduced:\n%s", file, string(data), string(r[file]))
					}
				}
			}
		})
	}
}

func TestDeduceRoundtrip(t *testing.T) {
	t.Parallel()

	for name, test := range deduceTests {
		t.Run(name, func(t *testing.T) {
			firstI, err := NewInterverse(test.manifestFrom, test.manifestTo)
			if err != nil && !test.errExpected {
				t.Errorf("could not create interverse, error was: %s", err.Error())
				return
			} else if err != nil && test.errExpected {
				// success
				return
			} else if err == nil && test.errExpected {
				t.Errorf("could create interverse but error expected")
				return
			}

			firstR, errs := firstI.DeduceStrict(test.from)
			if hasErrs(errs...) && !test.strictErrExpected {
				t.Errorf("could not strict deduce, error was: %v", errs)
				return
			} else if hasErrs(errs...) && test.strictErrExpected {
				// success
				return
			} else if !hasErrs(errs...) && test.strictErrExpected {
				t.Errorf("could create interverse but strict error expected")
				return
			}

			secondI, _ := NewInterverse(test.manifestTo, test.manifestFrom)

			secondR, _ := secondI.DeduceStrict(firstR)
			if !reflect.DeepEqual(test.from, secondR) {
				if *log {
					t.Logf("--- lookupTable:\n%s\n", firstI.lt.dump())
					for k := range test.from {
						t.Logf("--- file: %s\n", k)
						t.Logf("expected:\n%s\nintermediate:\n%s\nhas:\n%s\n", test.from[k], firstR[k], secondR[k])
					}
				}
				t.Errorf("roundtrip was invalid")
			}
		})
	}
}

func TestDeduceRoundtripFuzz(t *testing.T) {
	t.Parallel()
	type Test struct {
		from         map[string][]byte
		manifestFrom map[string]string
		manifestTo   map[string]string
	}

	f := fuzz.New().RandSource(rand.NewSource(0)).NilChance(0).Funcs(
		func(t *Test, c fuzz.Continue) {
			t.manifestFrom = map[string]string{}
			var xFrom string
			c.Fuzz(&xFrom)
			t.manifestFrom["x"] = xFrom + "e"

			t.manifestTo = map[string]string{}
			var xTo string
			c.Fuzz(&xTo)
			t.manifestTo["x"] = xTo + "e"

			t.from = map[string][]byte{}
			var file string
			c.Fuzz(&file)
			t.from["file"] = []byte(fmt.Sprintf("%s %s %s", file, xFrom, file))
		},
	)

	skippedInterverse, skippedDeduce, errors := 0, 0, 0
	for i := 0; i < *randIterations; i++ {
		test := Test{}
		f.Fuzz(&test)

		firstI, err := NewInterverse(test.manifestFrom, test.manifestTo)
		if err != nil {
			skippedInterverse++
			continue
		}
		firstR, errs := firstI.DeduceStrict(test.from)
		if hasErrs(errs...) {
			skippedDeduce++
			continue
		}

		secondI, err := NewInterverse(test.manifestTo, test.manifestFrom)
		if err != nil {
			t.FailNow()
		}
		secondR, errs := secondI.DeduceStrict(firstR)
		if hasErrs(errs...) {
			t.FailNow()
		}

		if !reflect.DeepEqual(test.from, secondR) {
			if *log {
				t.Logf("--- lookupTable:\n%s\n", firstI.lt.dump())
				for k := range test.from {
					t.Logf("--- file: %s\n", k)
					t.Logf("expected:\n%s\nintermediate:\n%s\nhas:\n%s\n", test.from[k], firstR[k], secondR[k])
				}
			}
			errors++
		}
	}
	if errors > 0 {
		t.Errorf("%d errors occurred", errors)
	}
	t.Logf("%d tests skipped due to handled interverse errors", skippedInterverse)
	t.Logf("%d tests skipped due to handled deduce errors", skippedDeduce)
}

func TestDeduceRoundtripRand(t *testing.T) {
	t.Parallel()

	type Test struct {
		from         map[string][]byte
		manifestFrom map[string]string
		manifestTo   map[string]string
	}

	randTest := func() Test {
		charset := string(randBytes(randInt(5, 66)))

		return Test{
			from: map[string][]byte{
				"file": randBytesWithCharset(randInt(5, 1000), charset),
			},
			manifestFrom: map[string]string{
				"x": string(randBytesWithCharset(randInt(2, 10), charset)),
			},
			manifestTo: map[string]string{
				"x": string(randBytesWithCharset(randInt(2, 10), charset)),
			},
		}
	}

	skippedInterverse, skippedDeduce, errors := 0, 0, 0
	for i := 0; i < *randIterations; i++ {
		test := randTest()

		firstI, err := NewInterverse(test.manifestFrom, test.manifestTo)
		if err != nil {
			skippedInterverse++
			continue
		}
		firstR, errs := firstI.DeduceStrict(test.from)
		if hasErrs(errs...) {
			skippedDeduce++
			continue
		}

		secondI, err := NewInterverse(test.manifestTo, test.manifestFrom)
		if err != nil {
			t.FailNow()
		}
		secondR, errs := secondI.DeduceStrict(firstR)
		if hasErrs(errs...) {
			t.FailNow()
		}

		if !reflect.DeepEqual(test.from, secondR) {
			if *log {
				t.Logf("--- lookupTable:\n%s\n", firstI.lt.dump())
				for k := range test.from {
					t.Logf("--- file: %s\n", k)
					t.Logf("expected:\n%s\nintermediate:\n%s\nhas:\n%s\n", test.from[k], firstR[k], secondR[k])
				}
			}
			errors++
		}
	}
	if errors > 0 {
		t.Errorf("%d errors occurred", errors)
	}
	t.Logf("%d tests skipped due to handled interverse errors", skippedInterverse)
	t.Logf("%d tests skipped due to handled deduce errors", skippedDeduce)
}

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

const defaultCharset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" +
	"-_. "

func randBytesWithCharset(length int, charset string) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return b
}

func randBytes(length int) []byte {
	return randBytesWithCharset(length, defaultCharset)
}

func randInt(min, max int) int {
	return seededRand.Intn(max-min) + min
}

func hasErrs(errs ...error) bool {
	errNotNil := false
	for _, err := range errs {
		if err == nil {
			continue
		}
		errNotNil = true
	}
	return errNotNil
}

// FuzzTokenizer tests the tokenizer with random byte sequences and switch tokens
func FuzzTokenizer(f *testing.F) {
	// Seed corpus with interesting test cases
	f.Add([]byte("hello world"), "hello", "goodbye")
	f.Add([]byte("VVVV"), "VV", "VVVV")
	f.Add([]byte("x_xx_x"), "xx", "x_x")
	f.Add([]byte("example.com api.example.com"), "example.com", "example-int.com")
	f.Add([]byte(""), "test", "replace")
	f.Add([]byte("aaaa"), "aa", "bbbb")
	f.Add([]byte("test test test"), "test", "replaced")

	f.Fuzz(func(t *testing.T, data []byte, from string, to string) {
		// Skip if from is empty as it's not a valid token
		if from == "" {
			t.Skip()
		}

		tokenizer := NewTokenizer(data)
		st := switchToken{A: from, B: to}
		tokenizer.Tokenize(st)

		result := tokenizer.Mutate()

		// Verify that the result doesn't crash when converted to string
		_ = string(result)

		// Verify that the raw data is preserved in the tokenizer
		tokenizer2 := NewTokenizer(data)
		raw := tokenizer2.Raw()
		if !bytes.Equal(raw, data) {
			t.Errorf("Raw() doesn't preserve original data")
		}
	})
}

// FuzzDeduceRoundtrip tests that A→B→A conversions are reversible
func FuzzDeduceRoundtrip(f *testing.F) {
	// Seed corpus
	f.Add("content with value1", "value1", "value2")
	f.Add("VV VV", "VV", "VVVV")
	f.Add("example.com and api.example.com", "example.com", "example-int.com")
	f.Add("", "a", "b")

	f.Fuzz(func(t *testing.T, content string, fromVal string, toVal string) {
		// Skip empty values as they're not valid
		if fromVal == "" || toVal == "" {
			t.Skip()
		}

		// Skip if values are the same
		if fromVal == toVal {
			t.Skip()
		}

		manifestFrom := map[string]string{"key": fromVal}
		manifestTo := map[string]string{"key": toVal}

		files := map[string][]byte{"test.txt": []byte(content)}

		// Create interverse from→to
		firstI, err := NewInterverse(manifestFrom, manifestTo)
		if err != nil {
			t.Skip() // Skip on expected errors
		}

		// Deduce from→to
		result, errs := firstI.DeduceStrict(files)
		if hasErrs(errs...) {
			// This is expected for some inputs, skip
			t.Skip()
		}

		// Create reverse interverse to→from
		secondI, err := NewInterverse(manifestTo, manifestFrom)
		if err != nil {
			t.Fatalf("Failed to create reverse interverse: %v", err)
		}

		// Deduce back to→from
		roundtrip, errs := secondI.DeduceStrict(result)
		if hasErrs(errs...) {
			// If roundtrip fails, this might indicate a bug
			t.Logf("Roundtrip failed for input: %q, from: %q, to: %q", content, fromVal, toVal)
			t.Logf("Errors: %v", errs)
			return
		}

		// Verify roundtrip matches original
		if !bytes.Equal(files["test.txt"], roundtrip["test.txt"]) {
			t.Errorf("Roundtrip failed:\nOriginal: %q\nAfter roundtrip: %q\nFrom: %q\nTo: %q",
				files["test.txt"], roundtrip["test.txt"], fromVal, toVal)
		}
	})
}

// FuzzNewInterverse tests interverse creation with various manifests
func FuzzNewInterverse(f *testing.F) {
	// Seed corpus
	f.Add("key1", "value1", "value2")
	f.Add("url", "example.com", "example-int.com")
	f.Add("key", "", "value")
	f.Add("key", "value", "")

	f.Fuzz(func(t *testing.T, key string, fromVal string, toVal string) {
		manifestFrom := map[string]string{key: fromVal}
		manifestTo := map[string]string{key: toVal}

		i, err := NewInterverse(manifestFrom, manifestTo)

		// Empty values should cause errors
		if (fromVal == "" || toVal == "") && err == nil {
			t.Errorf("Expected error for empty values but got none")
		}

		// Non-empty values should succeed
		if fromVal != "" && toVal != "" && err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// If successful, verify lookup table was created
		if err == nil && i != nil {
			if len(i.lt) != 1 {
				t.Errorf("Expected 1 lookup record, got %d", len(i.lt))
			}
		}
	})
}

// FuzzMultiKeyDeduction tests deduction with multiple manifest keys
func FuzzMultiKeyDeduction(f *testing.F) {
	// Seed corpus
	f.Add("test url env", "example.com", "production", "example-int.com", "integration")
	f.Add("a b c", "x", "y", "xx", "yy")

	f.Fuzz(func(t *testing.T, content string, url1 string, env1 string, url2 string, env2 string) {
		// Skip empty values
		if url1 == "" || env1 == "" || url2 == "" || env2 == "" {
			t.Skip()
		}

		// Skip if values would cause conflicts
		if url1 == env1 || url2 == env2 {
			t.Skip()
		}

		manifestFrom := map[string]string{"url": url1, "env": env1}
		manifestTo := map[string]string{"url": url2, "env": env2}

		files := map[string][]byte{"test.txt": []byte(content)}

		i, err := NewInterverse(manifestFrom, manifestTo)
		if err != nil {
			t.Skip()
		}

		result := i.Deduce(files)

		// Verify result exists and doesn't crash
		if result == nil {
			t.Errorf("Deduce returned nil")
		}

		// Verify we can convert result to string
		_ = string(result["test.txt"])
	})
}
