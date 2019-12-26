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

var randIterations = flag.Int("randiter", 1000, "number of iterations for randomized/fuzzed tests")
var log = flag.Bool("log", false, "print additional log")

func TestDeduce(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		manifestFrom    map[string]string
		from            map[string][]byte
		manifestTo      map[string]string
		to              map[string][]byte
		errExpected     bool
		toFoundExpected bool
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
			errExpected:     false,
			toFoundExpected: false,
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
			to:              map[string][]byte{},
			errExpected:     true,
			toFoundExpected: false,
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
			errExpected:     false,
			toFoundExpected: false,
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
			errExpected:     false,
			toFoundExpected: false,
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
			errExpected:     false,
			toFoundExpected: true,
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
			errExpected:     true,
			toFoundExpected: false,
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
			errExpected:     true,
			toFoundExpected: false,
		},
	}

	for name, test := range tests {
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

			r, toFound := i.Deduce(test.from)
			if !reflect.DeepEqual(r, test.to) {
				for file, data := range test.to {
					if !bytes.Equal(data, r[file]) {
						t.Errorf("result for file %s not as expected:\n--- Expected:\n%s\n--- Deduced:\n%s", file, string(data), string(r[file]))
					}
				}
			}
			hasToFound := len(toFound) > 0
			if hasToFound && !test.toFoundExpected {
				t.Errorf("found keys of destination alterverse but did not expect to, keys are: %s", toFound)
			} else if !hasToFound && test.toFoundExpected {
				t.Errorf("has not found any keys of destination alterverse but expected to")
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
			t.manifestFrom["x"] = xFrom

			t.manifestTo = map[string]string{}
			var xTo string
			c.Fuzz(&xTo)
			t.manifestTo["x"] = xTo

			t.from = map[string][]byte{}
			var file string
			c.Fuzz(&file)
			t.from["file"] = []byte(fmt.Sprintf("%s %s %s", file, xFrom, file))
		},
	)

	skipped, errors := 0, 0
	for i := 0; i < *randIterations; i++ {
		test := Test{}
		f.Fuzz(&test)

		firstI, err := NewInterverse(test.manifestFrom, test.manifestTo)
		if err != nil {
			skipped++
			continue
		}
		firstR, firstToFound := firstI.Deduce(test.from)

		secondI, err := NewInterverse(test.manifestTo, test.manifestFrom)
		if err != nil {
			skipped++
			continue
		}
		secondR, _ := secondI.Deduce(firstR)

		if !reflect.DeepEqual(test.from, secondR) && len(firstToFound) == 0 {
			if *log {
				t.Logf("--- lookupTable:\n%s\n", firstI.lt.dump())
				for k := range test.from {
					t.Logf("--- file: %s\n", k)
					t.Logf("expected:\n%s\nhas:\n%s\n", test.from[k], secondR[k])
				}
			}
			errors++
		}
	}
	if errors > 0 {
		t.Errorf("%d errors occurred", errors)
	}
	t.Logf("%d tests skipped due to handled errors", skipped)
}

func TestDeduceRoundtripRand(t *testing.T) {
	t.Parallel()
	type Test struct {
		from         map[string][]byte
		manifestFrom map[string]string
		manifestTo   map[string]string
	}

	randTest := func() Test {
		charset := string(randBytes(randInt(1, 66)))

		return Test{
			from: map[string][]byte{
				"file": randBytesWithCharset(randInt(5, 1000), charset),
			},
			manifestFrom: map[string]string{
				"x": string(randBytesWithCharset(randInt(1, 10), charset)),
			},
			manifestTo: map[string]string{
				"x": string(randBytesWithCharset(randInt(1, 10), charset)),
			},
		}
	}

	skipped, errors := 0, 0
	for i := 0; i < *randIterations; i++ {
		test := randTest()

		firstI, err := NewInterverse(test.manifestFrom, test.manifestTo)
		if err != nil {
			skipped++
			continue
		}
		firstR, firstToFound := firstI.Deduce(test.from)

		secondI, err := NewInterverse(test.manifestTo, test.manifestFrom)
		if err != nil {
			skipped++
			continue
		}
		secondR, _ := secondI.Deduce(firstR)

		if !reflect.DeepEqual(test.from, secondR) && len(firstToFound) == 0 {
			if *log {
				t.Logf("--- lookupTable:\n%s\n", firstI.lt.dump())
				for k := range test.from {
					t.Logf("--- file: %s\n", k)
					t.Logf("expected:\n%s\nhas:\n%s\n", test.from[k], secondR[k])
				}
			}
			errors++
		}
	}
	if errors > 0 {
		t.Errorf("%d errors occurred", errors)
	}
	t.Logf("%d tests skipped due to handled errors", skipped)
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
