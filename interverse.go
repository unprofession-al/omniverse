package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/xid"
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

// Deduce performs the actuall string substitution using the lookup table.
// To ensure no faulty substitutions occure every required value from the
// source is replaced with a generated key/id of a fixed length which in
// guarantied to be unique.
func (t Interverse) Deduce(in map[string][]byte) (out map[string][]byte, toFound map[string][]string) {
	intermediate := map[string][]byte{}
	for k, v := range in {
		data := v
		for _, lr := range t.lt {
			data = bytes.ReplaceAll(data, []byte(lr.From), []byte(lr.Key))
		}
		if !bytes.Equal(v, data) {
			intermediate[k] = data
		}
	}

	toFound = map[string][]string{}
	for k, v := range intermediate {
		for _, lr := range t.lt {
			if bytes.Contains(v, []byte(lr.To)) {
				if existing, ok := toFound[lr.To]; ok {
					toFound[lr.To] = append(existing, k)
				} else {
					toFound[lr.To] = []string{k}
				}
			}
		}
	}

	out = map[string][]byte{}
	for k, v := range in {
		out[k] = v
	}

	for k, v := range intermediate {
		data := v
		for _, lr := range t.lt {
			data = bytes.ReplaceAll(data, []byte(lr.Key), []byte(lr.To))
		}
		out[k] = data
	}

	return
}

type lookupRecord struct {
	From string
	To   string
	Key  string
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
			Key:  xid.New().String(),
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
