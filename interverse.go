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
