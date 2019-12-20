package main

import (
	"fmt"
	"os"
)

var (
	// These variables are passed during `go build` via ldflags, for example:
	//   go build -ldflags "-X main.commit=$(git rev-list -1 HEAD)"
	// goreleaser (https://goreleaser.com/) does this by default.
	version string
	commit  string
	date    string
)

func main() {
	err := rootCmd.Execute()
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

// verisonInfo returns a string containing information usually passed via
// ldflags during build time.
func versionInfo() string {
	if version == "" {
		version = "dirty"
	}
	if commit == "" {
		commit = "dirty"
	}
	if date == "" {
		date = "unknown"
	}
	return fmt.Sprintf("Version:    %s\nCommit:     %s\nBuild Date: %s\n", version, commit, date)
}
