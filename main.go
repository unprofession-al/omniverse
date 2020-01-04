package main

import (
	"fmt"
	"os"
)

func main() {
	err := NewApp().Execute()
	exitOnErr(err)
}

// exitOnErr takes an arbitary number of errors and prints those to stderr
// if they are not nil. If any non-nil errors where passed the program will
// be exited.
func exitOnErr(errs ...error) {
	errNotNil := false
	for _, err := range errs {
		if err == nil {
			continue
		}
		errNotNil = true
		fmt.Fprintf(os.Stderr, "ERROR: %s", err.Error())
	}
	if errNotNil {
		fmt.Print("\n")
		os.Exit(-1)
	}
}
