package main

import (
	"bytes"
	"testing"
)

func TestAlterverseLineReplacer(t *testing.T) {
	a := alterverse{
		definitions: map[string]string{
			"short": "https://www.foo.com",
			"long":  "https://www.foo.com/bar",
		},
	}

	expressionTemplate := `<< {{.}} >>`

	tests := []struct {
		desc    string
		in      []byte
		out     []byte
		changed bool
	}{
		{
			desc:    "nothing to replace",
			in:      []byte("There is nothing to change"),
			out:     []byte("There is nothing to change"),
			changed: false,
		},
		{
			desc:    "single string to replace",
			in:      []byte("This [link](https://www.foo.com) goes to a website"),
			out:     []byte("This [link](<< short >>) goes to a website"),
			changed: true,
		},
		{
			desc:    "multiple similar strings to replace",
			in:      []byte("This [link](https://www.foo.com) goes to https://www.foo.com"),
			out:     []byte("This [link](<< short >>) goes to << short >>"),
			changed: true,
		},
		{
			desc:    "multiple strings, long first",
			in:      []byte("This [link](https://www.foo.com/bar) does not go to https://www.foo.com"),
			out:     []byte("This [link](<< long >>) does not go to << short >>"),
			changed: true,
		},
		{
			desc:    "multiple strings, short first",
			in:      []byte("This [link](https://www.foo.com) does not go to https://www.foo.com/bar"),
			out:     []byte("This [link](<< short >>) does not go to << long >>"),
			changed: true,
		},
		{
			desc:    "multiple strings, short first, long is mismatch",
			in:      []byte("This [link](https://www.foo.com) does not go to https://www.foo.com/BAR"),
			out:     []byte("This [link](<< short >>) does not go to << short >>/BAR"),
			changed: true,
		},
	}

	lr, err := a.GetLineReplacer(expressionTemplate)
	if err != nil {
		t.Errorf("error while generating line replacer: %s", err.Error())
	}
	for _, test := range tests {
		out, changed := lr(test.in)
		if changed != test.changed || !bytes.Equal(out, test.out) {
			t.Errorf("faulty replacement for case '%s':\n\tshould be: %s\n\tis      : %s", test.desc, test.out, out)
		}

	}
}
