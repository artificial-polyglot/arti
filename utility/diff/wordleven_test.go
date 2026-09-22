package diff

import (
	"math"
	"strings"
	"testing"
)

// wordDistance is an independent word-level Levenshtein distance, to cross-check
// that the aligner's op count is minimal without pinning an exact alignment.
func wordDistance(a, b string) int {
	ar, br := strings.Fields(a), strings.Fields(b)
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

// reconstructWords rebuilds the ref and hyp word sequences from a word-level
// alignment (Text is the ref word except on Insert, where it is the asr word).
func reconstructWords(diffs []Diff) (ref, hyp []string) {
	for _, d := range diffs {
		switch d.Type {
		case OpEqual:
			ref = append(ref, d.Text)
			hyp = append(hyp, d.Text)
		case OpReplace:
			ref = append(ref, d.Text)
			hyp = append(hyp, d.Replace)
		case OpDelete:
			ref = append(ref, d.Text)
		case OpInsert:
			hyp = append(hyp, d.Text)
		}
	}
	return
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestWordLevenshtein(t *testing.T) {
	tests := []struct {
		name     string
		ref, asr string
		want     []Diff // nil = ambiguous alignment; invariants only
	}{
		{
			name: "all equal", ref: "the cat sat", asr: "the cat sat",
			want: []Diff{{OpEqual, "the", ""}, {OpEqual, "cat", ""}, {OpEqual, "sat", ""}},
		},
		{
			name: "one word substituted", ref: "the quick brown fox", asr: "the fast brown fox",
			want: []Diff{
				{OpEqual, "the", ""},
				{OpReplace, "quick", "fast"},
				{OpEqual, "brown", ""},
				{OpEqual, "fox", ""},
			},
		},
		{
			name: "word deleted", ref: "a b c", asr: "a c",
			want: []Diff{{OpEqual, "a", ""}, {OpDelete, "b", ""}, {OpEqual, "c", ""}},
		},
		{
			name: "word inserted", ref: "a c", asr: "a b c",
			want: []Diff{{OpEqual, "a", ""}, {OpInsert, "b", ""}, {OpEqual, "c", ""}},
		},
		{
			name: "all substituted", ref: "the cat sat", asr: "a dog ran",
			want: []Diff{
				{OpReplace, "the", "a"},
				{OpReplace, "cat", "dog"},
				{OpReplace, "sat", "ran"},
			},
		},
		{
			name: "collapsed whitespace is ignored", ref: "  the   cat ", asr: "the cat",
			want: []Diff{{OpEqual, "the", ""}, {OpEqual, "cat", ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WordLevenshtein(tt.ref, tt.asr)

			// Invariant 1: reconstructs both word sequences.
			gotRef, gotHyp := reconstructWords(got)
			if !sliceEq(gotRef, strings.Fields(tt.ref)) || !sliceEq(gotHyp, strings.Fields(tt.asr)) {
				t.Fatalf("reconstruction failed: ref=%v asr=%v", gotRef, gotHyp)
			}
			// Invariant 2: op count == independent word edit distance (optimal).
			c := CountWords(got)
			if edits := c.Substitutions + c.Deletions + c.Insertions; edits != wordDistance(tt.ref, tt.asr) {
				t.Fatalf("op count %d != word distance %d\n got: %v", edits, wordDistance(tt.ref, tt.asr), got)
			}
			if tt.want != nil && !equalDiffs(got, tt.want) {
				t.Errorf("WordLevenshtein(%q,%q)\n got: %v\nwant: %v", tt.ref, tt.asr, got, tt.want)
			}
		})
	}
}

func TestWER(t *testing.T) {
	// ref has 4 words; quick->fast is one substitution => WER = 1/4 = 0.25.
	c := CountWords(WordLevenshtein("the quick brown fox", "the fast brown fox"))
	if c.Substitutions != 1 || c.Deletions != 0 || c.Insertions != 0 || c.RefLen != 4 {
		t.Fatalf("counts = %+v", c)
	}
	if math.Abs(c.ErrorRate()-0.25) > 1e-9 {
		t.Fatalf("WER = %v, want 0.25", c.ErrorRate())
	}

	// Fully wrong 3-word verse: 3 subs over 3 ref words => WER = 1.0.
	c2 := CountWords(WordLevenshtein("the cat sat", "a dog ran"))
	if math.Abs(c2.ErrorRate()-1.0) > 1e-9 {
		t.Fatalf("WER = %v, want 1.0", c2.ErrorRate())
	}
}
