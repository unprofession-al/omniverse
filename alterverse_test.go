package main

import "fmt"

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
