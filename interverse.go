package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

// Interverse holds all data and logic to convert the data from
// one alterverse to another alterverse.
type Interverse struct {
	lt lookupTable
}

// NewInterverse takes two manifests, builds a lookup table, sorts
// this table (long values of the 'source' alterverse must come first
// to ensure proper string substitution) and returns a ready to use
// Interverse.
func NewInterverse(from, to Manifest) (*Interverse, error) {
	i := &Interverse{}
	lt, err := newLookupTable(from, to)
	// the reverse sort is important: it ensures that long strings are replaced
	// first so shorter strings which are substrings of the longer ones do not
	// interfer with those.
	sort.Sort(sort.Reverse(lt))
	i.lt = lt
	return i, err
}

// Deduce performs the actual string substitution using the lookup table.
// This is done using the Tokenizer. Deduce can produce an alterverse that
// cannot be converted back to its source alterverse. To avoid this make
// use of the DeduceStrict method.
func (t Interverse) Deduce(in map[string][]byte) map[string][]byte {
	out := map[string][]byte{}
	for k, v := range in {
		tokenizer := NewTokenizer(v)
		for _, lr := range t.lt {
			st := switchToken{A: lr.From, B: lr.To}
			tokenizer.Tokenize(st)
		}
		out[k] = tokenizer.Mutate()
	}
	return out
}

// DeduceStrict performs the actual string substitution using the lookup table.
// This is done using the Tokenizer. DeduceStrict produces alterverses that
// can be converted back to its source alterverse but has a huge overhead compared
// to the Deduce method.
func (t Interverse) DeduceStrict(in map[string][]byte) (map[string][]byte, []error) {
	tokenizers := map[string]Tokenizer{}
	for k, v := range in {
		tokenizer := NewTokenizer(v)
		for _, lr := range t.lt {
			st := switchToken{A: lr.From, B: lr.To}
			tokenizer.Tokenize(st)
		}
		tokenizers[k] = tokenizer
	}

	out := map[string][]byte{}
	for k, v := range tokenizers {
		out[k] = v.Mutate()
	}

	errs := []error{}
	for k, v := range tokenizers {
		for _, lr := range t.lt {
			if v.Contains([]byte(lr.To)) {
				errs = append(errs, fmt.Errorf("file '%s' contains the string '%s' which is "+
					"the value of the manifest key '%s' of the destination alterverse", k, lr.To, lr.Name))
			}
		}
	}
	if len(errs) > 0 {
		return out, errs
	}

	reverse := map[string][]byte{}
	for k, v := range out {
		tokenizer := NewTokenizer(v)
		for _, lr := range t.lt {
			st := switchToken{A: lr.To, B: lr.From}
			tokenizer.Tokenize(st)
		}
		reverse[k] = tokenizer.Mutate()
	}

	for k := range in {
		if !bytes.Equal(in[k], reverse[k]) {
			errs = append(errs, fmt.Errorf("full monty failed for '%s'", k))
		}
	}

	return out, errs
}

type lookupRecord struct {
	From string
	To   string
	Name string
}

type lookupTable []*lookupRecord

func newLookupTable(from, to map[string]string) (lookupTable, error) {
	lt := []*lookupRecord{}

	if ok, missing := haveSameKeys(from, to); !ok {
		return lookupTable(lt), fmt.Errorf("the following keys are missing: %s", strings.Join(missing, ", "))
	}

	for k := range from {
		if from[k] == "" {
			return lookupTable(lt), fmt.Errorf("key	'%s' in 'from' manifest must not be empty", k)
		}

		if to[k] == "" {
			return lookupTable(lt), fmt.Errorf("key	'%s' in 'to' manifest must not be empty", k)
		}

		lr := &lookupRecord{
			From: from[k],
			To:   to[k],
			Name: k,
		}
		lt = append(lt, lr)
	}

	return lookupTable(lt), nil
}

// haveSameKeys checks two maps a and b if all keys present in a are also
// present in b (not vice versa!). A list of missing keys is returned as second
// return value.
func haveSameKeys(a, b map[string]string) (bool, []string) {
	missing := []string{}
	for k := range a {
		if _, ok := b[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return false, missing
	}
	return true, missing
}

func (lt lookupTable) Len() int           { return len(lt) }
func (lt lookupTable) Less(i, j int) bool { return len(lt[i].From) < len(lt[j].From) }
func (lt lookupTable) Swap(i, j int)      { lt[i], lt[j] = lt[j], lt[i] }

func (lt lookupTable) dump() string {
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "name\tfrom\tto")
	for _, record := range lt {
		fmt.Fprintf(w, "'%s'\t'%s'\t'%s'\n", record.Name, record.From, record.To)
	}
	w.Flush()
	return out.String()
}

// Tokenizer allow no take a split a byte slice and split it into tokens where
// token is an interface. This allows to replace parts of a byte slice with elements
// of the lookup table
type Tokenizer struct {
	tokens []token
}

func NewTokenizer(rawBytes []byte) Tokenizer {
	return Tokenizer{tokens: []token{byteToken(rawBytes)}}
}

func (t *Tokenizer) Tokenize(by token) {
	tmp := []token{}
	for _, token := range t.tokens {
		tmp = append(tmp, token.tokenize(by)...)
	}
	t.tokens = tmp
}

func (t *Tokenizer) Raw() []byte {
	out := []byte{}
	for _, token := range t.tokens {
		out = append(out, token.raw()...)
	}
	return out
}

func (t *Tokenizer) Mutate() []byte {
	out := []byte{}
	for _, token := range t.tokens {
		out = append(out, token.mutate()...)
	}
	return out
}

func (t *Tokenizer) Contains(b []byte) bool {
	for _, token := range t.tokens {
		if token.contains(b) {
			return true
		}
	}
	return false
}

func (t *Tokenizer) Dump() string {
	out := ""
	for _, token := range t.tokens {
		out = fmt.Sprintf("%s'%s' is of kind %s\n", out, string(token.raw()), token.kind())
	}
	return out
}

type token interface {
	tokenize(token) []token
	raw() []byte
	kind() string
	mutate() []byte
	contains([]byte) bool
}

type byteToken []byte

func (bt byteToken) tokenize(by token) []token {
	split := bytes.SplitN(bt, by.raw(), 2)

	noMatch := len(split) == 1
	endsWithMatch := len(split) == 2 && len(split[0]) > 0 && len(split[1]) == 0
	startsWithMatch := len(split) == 2 && len(split[0]) == 0 && len(split[1]) > 0
	innerMatch := len(split) == 2 && len(split[0]) > 0 && len(split[1]) > 0
	fullMatch := len(split) == 2 && len(split[0]) == 0 && len(split[1]) == 0

	var out []token
	if noMatch {
		out = []token{bt}
	} else if endsWithMatch {
		out = []token{byteToken(split[0]), by}
	} else if startsWithMatch {
		next := byteToken(split[1])
		out = append([]token{by}, next.tokenize(by)...)
	} else if innerMatch {
		next := byteToken(split[1])
		out = append([]token{byteToken(split[0]), by}, next.tokenize(by)...)
	} else if fullMatch {
		out = []token{by}
	}

	return out
}

func (bt byteToken) contains(b []byte) bool { return bytes.Contains(bt, b) }
func (bt byteToken) raw() []byte            { return []byte(bt) }
func (bt byteToken) mutate() []byte         { return bt.raw() }
func (bt byteToken) kind() string           { return "byteToken" }

type switchToken struct {
	A string
	B string
}

func (st switchToken) tokenize(by token) []token { return []token{st} }
func (st switchToken) contains(b []byte) bool    { return false }
func (st switchToken) raw() []byte               { return []byte(st.A) }
func (st switchToken) mutate() []byte            { return []byte(st.B) }
func (st switchToken) kind() string              { return "switchToken" }
