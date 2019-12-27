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
