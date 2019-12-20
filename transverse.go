package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/xid"
)

type Transverse struct {
	lt LookupTable
}

func NewTransverse(from, to Manifest) (*Transverse, error) {
	i := &Transverse{}
	lt, err := NewLookupTable(from, to)
	sort.Sort(sort.Reverse(lt))
	i.lt = lt
	return i, err
}

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

type LookupRecord struct {
	From string
	To   string
	Key  string
	Name string
}

type LookupTable []*LookupRecord

func NewLookupTable(from, to map[string]string) (LookupTable, error) {
	lt := []*LookupRecord{}

	if ok, missing := haveSameKeys(from, to); !ok {
		return LookupTable(lt), fmt.Errorf("the following keys are missing: %s", strings.Join(missing, ", "))
	}

	for k, v := range from {
		lr := &LookupRecord{
			From: v,
			To:   to[k],
			Key:  xid.New().String(),
			Name: k,
		}
		lt = append(lt, lr)
	}

	return LookupTable(lt), nil
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

func (lt LookupTable) Len() int           { return len(lt) }
func (lt LookupTable) Less(i, j int) bool { return len(lt[i].From) < len(lt[j].From) }
func (lt LookupTable) Swap(i, j int)      { lt[i], lt[j] = lt[j], lt[i] }
