package diff

import (
	"testing"
)

// --- DiffReplace -----------------------------------------------------------

func TestDiffReplace(t *testing.T) {
	tests := []struct {
		name     string
		ref, asr string
		want     []Diff
	}{
		{
			name: "equal only",
			ref:  "hello", asr: "hello",
			want: []Diff{{Type: OpEqual, Text: "hello"}},
		},
		{
			name: "pure insert",
			ref:  "abc", asr: "abcd",
			want: []Diff{{Type: OpEqual, Text: "abc"}, {Type: OpInsert, Text: "d"}},
		},
		{
			name: "trailing delete is not dropped",
			ref:  "abc def", asr: "abc",
			want: []Diff{{Type: OpEqual, Text: "abc"}, {Type: OpDelete, Text: " def"}},
		},
		{
			name: "equal-length replace",
			ref:  "abc", asr: "xyz",
			want: []Diff{{Type: OpReplace, Text: "abc", Replace: "xyz"}},
		},
		{
			name: "multibyte replace, equal runes",
			ref:  "café", asr: "cafe",
			want: []Diff{{Type: OpEqual, Text: "caf"}, {Type: OpReplace, Text: "é", Replace: "e"}},
		},
		{
			name: "ref longer than asr: replace + delete remainder",
			ref:  "abc", asr: "aé",
			want: []Diff{
				{Type: OpEqual, Text: "a"},
				{Type: OpReplace, Text: "b", Replace: "é"},
				{Type: OpDelete, Text: "c"},
			},
		},
		{
			name: "asr longer than ref, ref multibyte: replace + insert remainder",
			ref:  "é", asr: "ab",
			want: []Diff{
				{Type: OpReplace, Text: "é", Replace: "a"},
				{Type: OpInsert, Text: "b"},
			},
		},
		{
			name: "ref longer than asr, asr multibyte: replace + delete remainder",
			ref:  "ab", asr: "é",
			want: []Diff{
				{Type: OpReplace, Text: "a", Replace: "é"},
				{Type: OpDelete, Text: "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiffReplace(tt.ref, tt.asr)
			if !equalDiffs(got, tt.want) {
				t.Errorf("DiffReplace(%q,%q)\n got: %v\nwant: %v", tt.ref, tt.asr, got, tt.want)
			}
		})
	}
}

// --- CharDiff --------------------------------------------------------------

func TestCharDiff(t *testing.T) {
	tests := []struct {
		name     string
		ref, asr string
		want     []CDiff
	}{
		{
			name: "equal-length replace pairs each rune",
			ref:  "abc", asr: "xyz",
			want: []CDiff{
				{Type: OpReplace, Char: 'a', Replace: 'x'},
				{Type: OpReplace, Char: 'b', Replace: 'y'},
				{Type: OpReplace, Char: 'c', Replace: 'z'},
			},
		},
		{
			name: "multibyte replace does not panic or corrupt",
			ref:  "café", asr: "cafe",
			want: []CDiff{
				{Type: OpEqual, Char: 'c'},
				{Type: OpEqual, Char: 'a'},
				{Type: OpEqual, Char: 'f'},
				{Type: OpReplace, Char: r("é"), Replace: 'e'},
			},
		},
		{
			name: "asr longer, ref multibyte: replace then insert remainder",
			ref:  "é", asr: "ab",
			want: []CDiff{
				{Type: OpReplace, Char: r("é"), Replace: 'a'},
				{Type: OpInsert, Char: 'b'},
			},
		},
		{
			name: "ref longer, asr multibyte: replace then delete remainder",
			ref:  "ab", asr: "é",
			want: []CDiff{
				{Type: OpReplace, Char: 'a', Replace: r("é")},
				{Type: OpDelete, Char: 'b'},
			},
		},
		{
			name: "equal and deleted spaces are dropped",
			ref:  "a b", asr: "a c",
			want: []CDiff{
				{Type: OpEqual, Char: 'a'},
				{Type: OpReplace, Char: 'b', Replace: 'c'},
			},
		},
		{
			name: "inserted spaces are kept",
			ref:  "ab", asr: "a b",
			want: []CDiff{
				{Type: OpEqual, Char: 'a'},
				{Type: OpInsert, Char: ' '},
				{Type: OpEqual, Char: 'b'},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CharDiff(tt.ref, tt.asr)
			if !equalCDiffs(got, tt.want) {
				t.Errorf("CharDiff(%q,%q)\n got: %v\nwant: %v", tt.ref, tt.asr, got, tt.want)
			}
		})
	}
}

// --- helpers ---------------------------------------------------------------

func equalDiffs(a, b []Diff) bool {
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

func equalCDiffs(a, b []CDiff) bool {
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

// r is a shorthand for a single rune from a one-rune string literal.
func r(s string) rune { return []rune(s)[0] }
