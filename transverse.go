package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/xid"
)

// Transverse holds all data and logic to convert the data from
// one alterverse to anotherp.
type Transverse struct {
	lt lookupTable
}

// NewTransverse takes two manifests, builds a lookup table, sorts
// this table (long values of the 'source' alterverse must come first
// to ensure proper string substitution) and returns a ready to use
// Transverse
func NewTransverse(from, to Manifest) (*Transverse, error) {
	i := &Transverse{}
	lt, err := newLookupTable(from, to)
	sort.Sort(sort.Reverse(lt))
	i.lt = lt
	return i, err
}

// Do performs the actuall string substitution using the lookup table.
// To ensure no faulty substitutions occure every required value from the
// source is replaced with a generated key/id of a fixed length which in
// guarantied to be unique.
func (t Transverse) Do(in map[string][]byte) map[string][]byte {
	intermediate := map[string][]byte{}
	for k, v := range in {
		data := v
		for _, lr := range t.lt {
			data = bytes.ReplaceAll(data, []byte(lr.From), []byte(lr.Key))
		}
		if !bytes.Equal(v, data) {
			intermediate[k] = data
		} else {
			fmt.Printf("file '%s' is unchanged\n", k)
		}
	}

	out := in
	for k, v := range intermediate {
		fmt.Printf("file '%s' is changed\n", k)
		data := v
		for _, lr := range t.lt {
			data = bytes.ReplaceAll(data, []byte(lr.Key), []byte(lr.To))
		}
		out[k] = data
	}

	return out
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

	for k, v := range from {
		lr := &lookupRecord{
			From: v,
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
