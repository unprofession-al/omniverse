package main

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func DiffFiles(a, b map[string][]byte) (diffs map[string]string, obsolete, created map[string][]byte) {
	common, obsolete, created := findCommonFiles(a, b)

	diffs = map[string]string{}
	dmp := diffmatchpatch.New()
	for k := range common {
		dataA := string(a[k])
		dataB := string(b[k])

		diff := dmp.DiffMain(dataA, dataB, false)
		diffs[k] = getLineDiff(diff, dmp)
	}

	return
}

func getLineDiff(diff []diffmatchpatch.Diff, dmp *diffmatchpatch.DiffMatchPatch) string {
	out := ""
	if len(diff) > 1 {
		r := strings.NewReader(dmp.DiffPrettyText(diff))
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			// if the line contains any collor markers as used in
			// https://github.com/sergi/go-diff/blob/master/diffmatchpatch/diff.go#L1183
			// we consider the line to contain a change
			if strings.Contains(line, "\x1b[32m") ||
				strings.Contains(line, "\x1b[31m") ||
				strings.Contains(line, "\x1b[0m") {
				out = fmt.Sprintf("%s%s\n", out, line)
			}
		}
		if err := scanner.Err(); err != nil {
			err = nil
			// TODO: we ignore that for because its just for debug output
			// however this obvously shoud be handled
		}
	}
	return out
}
