package diff

import (
	"math"
	"testing"
)

// levDistance is an independent (rolling-array) Levenshtein distance used to
// cross-check that the aligner's op count is truly minimal, without pinning a
// specific alignment when several are equally optimal.
func levDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	n, m := len(ar), len(br)
	prev := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}
	for i := 1; i <= n; i++ {
		cur := make([]int, m+1)
		cur[0] = i
		for j := 1; j <= m; j++ {
			c := 1
			if ar[i-1] == br[j-1] {
				c = 0
			}
			cur[j] = min(prev[j-1]+c, min(prev[j]+1, cur[j-1]+1))
		}
		prev = cur
	}
	return prev[m]
}

// reconstruct rebuilds (ref, hyp) from an alignment. Every op carries its single
// rune in Char, except OpReplace which carries the ref rune in Char and the asr
// rune in Replace — matching the existing CDiff convention.
func reconstruct(diffs []CDiff) (ref, hyp string) {
	var rb, hb []rune
	for _, d := range diffs {
		switch d.Type {
		case OpEqual:
			rb = append(rb, d.Char)
			hb = append(hb, d.Char)
		case OpReplace:
			rb = append(rb, d.Char)
			hb = append(hb, d.Replace)
		case OpDelete:
			rb = append(rb, d.Char)
		case OpInsert:
			hb = append(hb, d.Char)
		}
	}
	return string(rb), string(hb)
}

func TestCharLevenshtein(t *testing.T) {
	tests := []struct {
		name     string
		ref, asr string
		want     []CDiff // nil = alignment is ambiguous; check invariants only
	}{
		{
			name: "equal", ref: "abc", asr: "abc",
			want: []CDiff{{OpEqual, 'a', 0}, {OpEqual, 'b', 0}, {OpEqual, 'c', 0}},
		},
		{
			name: "all substitutions", ref: "abc", asr: "xyz",
			want: []CDiff{{OpReplace, 'a', 'x'}, {OpReplace, 'b', 'y'}, {OpReplace, 'c', 'z'}},
		},
		{
			name: "single substitution in the middle", ref: "cat", asr: "cot",
			want: []CDiff{{OpEqual, 'c', 0}, {OpReplace, 'a', 'o'}, {OpEqual, 't', 0}},
		},
		{
			name: "multibyte substitution stays one op", ref: "café", asr: "cafe",
			want: []CDiff{{OpEqual, 'c', 0}, {OpEqual, 'a', 0}, {OpEqual, 'f', 0}, {OpReplace, r("é"), 'e'}},
		},
		{
			name: "pure deletion", ref: "abc", asr: "ac",
			want: []CDiff{{OpEqual, 'a', 0}, {OpDelete, 'b', 0}, {OpEqual, 'c', 0}},
		},
		{
			name: "pure insertion", ref: "ac", asr: "abc",
			want: []CDiff{{OpEqual, 'a', 0}, {OpInsert, 'b', 0}, {OpEqual, 'c', 0}},
		},
		// Ambiguous alignments (a substitution can trade places with an indel):
		// only the invariants are asserted.
		{name: "ref shorter, multibyte", ref: "é", asr: "ab"},
		{name: "ref longer, multibyte", ref: "ab", asr: "é"},
		{name: "unrelated words, trailing shared letter", ref: "three", asr: "five"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CharLevenshtein(tt.ref, tt.asr)

			// Invariant 1: the alignment reconstructs both inputs exactly.
			if gotRef, gotHyp := reconstruct(got); gotRef != tt.ref || gotHyp != tt.asr {
				t.Fatalf("reconstruction failed: ref=%q asr=%q", gotRef, gotHyp)
			}
			// Invariant 2: op count equals the true edit distance (optimality).
			c := Count(got)
			if edits := c.Substitutions + c.Deletions + c.Insertions; edits != levDistance(tt.ref, tt.asr) {
				t.Fatalf("op count %d != edit distance %d\n got: %v", edits, levDistance(tt.ref, tt.asr), got)
			}
			// Exact alignment where it is unambiguous.
			if tt.want != nil && !equalCDiffs(got, tt.want) {
				t.Errorf("CharLevenshtein(%q,%q)\n got: %v\nwant: %v", tt.ref, tt.asr, got, tt.want)
			}
		})
	}
}

func TestCountsAndErrorRate(t *testing.T) {
	// "kitten" -> "sitting": k->s, e->i (subs), insert g => S=2, I=1, D=0.
	c := Count(CharLevenshtein("kitten", "sitting"))
	if c.Substitutions != 2 || c.Insertions != 1 || c.Deletions != 0 {
		t.Fatalf("kitten/sitting counts = %+v", c)
	}
	if c.RefLen != 6 {
		t.Fatalf("RefLen = %d, want 6", c.RefLen)
	}
	if math.Abs(c.ErrorRate()-0.5) > 1e-9 { // (2+1+0)/6
		t.Fatalf("ErrorRate = %v, want 0.5", c.ErrorRate())
	}
}
