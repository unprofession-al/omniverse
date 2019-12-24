package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDeduce(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		manifestFrom map[string]string
		from         map[string][]byte
		manifestTo   map[string]string
		to           map[string][]byte
		errExpected  bool
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
			errExpected: false,
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
			to:          map[string][]byte{},
			errExpected: true,
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
			errExpected: false,
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
			errExpected: false,
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
