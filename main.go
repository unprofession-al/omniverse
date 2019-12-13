package main

import (
	"fmt"
	"os"
)

func main() {
	err := rootCmd.Execute()
	exitOnErr(err)
}

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
		os.Exit(-1)
	}
}
