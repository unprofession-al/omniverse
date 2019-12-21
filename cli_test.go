package main

import (
	"regexp"
	"testing"
)

func TestDefaultIgrore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		filename      string
		matchExpected bool
	}{
		{filename: "/etc/test", matchExpected: false},
		{filename: "etc/test", matchExpected: false},
		{filename: "test", matchExpected: false},
		{filename: "test.txt", matchExpected: false},
		{filename: "foo/test.txt", matchExpected: false},
		{filename: "foo.bar/test.txt", matchExpected: false},
		{filename: `foo\test.txt`, matchExpected: false},
		{filename: `c:\\foo.bar\test.txt`, matchExpected: false},
		{filename: "/var/lib/.teet", matchExpected: true},
		{filename: "mla/.test", matchExpected: true},
		{filename: ".test", matchExpected: true},
		{filename: `c:\\bsa/.sath`, matchExpected: true},
		{filename: `aoeu\.tsaoe`, matchExpected: true},
	}

	re := regexp.MustCompile(defaultIgrore)

	for _, test := range tests {
		t.Run("file "+test.filename, func(t *testing.T) {
			if re.MatchString(test.filename) && !test.matchExpected {
				t.Errorf("regexp `%s` match string '%s' but should not", defaultIgrore, test.filename)
			} else if !re.MatchString(test.filename) && test.matchExpected {
				t.Errorf("regexp `%s` did not match string '%s' but should", defaultIgrore, test.filename)
			}
		})
	}
}
