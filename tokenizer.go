package main

import (
	"bytes"
	"fmt"
)

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
