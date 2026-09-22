package diff

import "strings"

// WordLevenshtein aligns refText (reference) against asrText (hypothesis) at the
// WORD level using Levenshtein DP where substitution, insertion and deletion
// each cost 1. This is the standard basis for Word Error Rate (WER).
//
// Words are split with strings.Fields, so leading/trailing whitespace is
// ignored and runs of whitespace collapse; the alignment is over words only, so
// exact spacing is not reconstructable from the result (use the character path
// when you need that). Normalize (lowercase, strip punctuation, etc.) BEFORE
// calling if you want those handled — this compares whole words verbatim.
//
// It emits []Diff, matching the DiffReplace/CharDiff convention: Text holds the
// reference word for Equal/Replace/Delete and the ASR word for Insert; Replace
// holds the ASR word on a Replace.
func WordLevenshtein(refText string, asrText string) []Diff {
	ref := strings.Fields(refText)
	hyp := strings.Fields(asrText)
	n := len(ref)
	m := len(hyp)

	// cost[i][j] = word edit distance between ref[:i] and hyp[:j].
	cost := make([][]int, n+1)
	for i := range cost {
		cost[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		cost[i][0] = i
	}
	for j := 1; j <= m; j++ {
		cost[0][j] = j
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			sub := 1
			if ref[i-1] == hyp[j-1] {
				sub = 0
			}
			cost[i][j] = min3(
				cost[i-1][j-1]+sub, // substitute or match
				cost[i-1][j]+1,     // delete ref word
				cost[i][j-1]+1,     // insert hyp word
			)
		}
	}

	// Backtrace, preferring diagonal (match/sub), then delete, then insert.
	var rev []Diff
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 {
			sub := 1
			if ref[i-1] == hyp[j-1] {
				sub = 0
			}
			if cost[i][j] == cost[i-1][j-1]+sub {
				if sub == 0 {
					rev = append(rev, Diff{Type: OpEqual, Text: ref[i-1]})
				} else {
					rev = append(rev, Diff{Type: OpReplace, Text: ref[i-1], Replace: hyp[j-1]})
				}
				i--
				j--
				continue
			}
		}
		if i > 0 && cost[i][j] == cost[i-1][j]+1 {
			rev = append(rev, Diff{Type: OpDelete, Text: ref[i-1]})
			i--
			continue
		}
		rev = append(rev, Diff{Type: OpInsert, Text: hyp[j-1]})
		j--
	}

	result := make([]Diff, len(rev))
	for k := range rev {
		result[k] = rev[len(rev)-1-k]
	}
	return result
}

// CountWords tallies a word-level []Diff into S/D/I/hits. Use the resulting
// Counts.ErrorRate() to get WER: (S+D+I)/RefLen.
func CountWords(diffs []Diff) Counts {
	var c Counts
	for _, d := range diffs {
		switch d.Type {
		case OpEqual:
			c.Hits++
		case OpReplace:
			c.Substitutions++
		case OpDelete:
			c.Deletions++
		case OpInsert:
			c.Insertions++
		}
	}
	c.RefLen = c.Hits + c.Substitutions + c.Deletions
	return c
}
