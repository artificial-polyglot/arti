package diff

import (
	"unicode/utf8"
)

// Counts holds the S/D/I/hits breakdown of an alignment.
type Counts struct {
	Hits          int // equal
	Substitutions int // replace
	Deletions     int // delete (in reference, not in hypothesis)
	Insertions    int // insert (in hypothesis, not in reference)
	RefLen        int // reference tokens = Hits + Substitutions + Deletions
}

func Count(diffs []CDiff) Counts {
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

func CountDiff(diffs []Diff) Counts {
	var c Counts
	for _, d := range diffs {
		length := utf8.RuneCountInString(d.Text)
		switch d.Type {
		case OpEqual:
			c.Hits += length
		case OpReplace:
			c.Substitutions += length
		case OpDelete:
			c.Deletions += length
		case OpInsert:
			c.Insertions += length
		}
	}
	c.RefLen = c.Hits + c.Substitutions + c.Deletions
	return c
}

// ErrorRate returns (S+D+I)/RefLen — CER when the tokens are runes. Returns 0
// for an empty reference.
func (c *Counts) ErrorRate() float64 {
	if c.RefLen == 0 {
		return 0
	}
	return float64(c.Substitutions+c.Deletions+c.Insertions) / float64(c.RefLen)
}

func (c *Counts) ErrorCount() int {
	return c.Substitutions + c.Deletions + c.Insertions
}

func (c *Counts) Add(cnt Counts) {
	c.Hits += cnt.Hits
	c.Substitutions += cnt.Substitutions
	c.Deletions += cnt.Deletions
	c.Insertions += cnt.Insertions
	c.RefLen += cnt.RefLen
}

func ErrorCount(diffs []CDiff) int {
	var count int
	for _, c := range diffs {
		switch c.Type {
		case OpEqual:
			//
		case OpInsert:
			if c.Char != ' ' && !IsHyphen(c.Char) && !IsLikeApostrophe(c.Char) {
				count++
			}
		case OpDelete:
			if c.Char != ' ' && !IsLikeApostrophe(c.Char) {
				count++
			}
		case OpReplace:
			count++
		}
	}
	return count
}

func IsLikeApostrophe(ch rune) bool {
	switch ch {
	case '\u0027':
		return true // apostrophe
	case '\u2018', '\u2019':
		return true // single quote marks
	case '\uA78C':
		return true // saltillo
	case '\u02BC':
		return true // glottal stop
	case '\u02BB':
		return true // Hawaian okina
	}
	return false
}

func IsHyphen(ch rune) bool {
	switch ch {
	case '\u002D', // hyphen-minus
		'\u00AD',                                                   // soft hyphen
		'\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015', // hyphen .. horizontal bar
		'\u2212',                     // minus sign
		'\uFE58', '\uFE62', '\uFE63', // small em dash, small hyphen-minus
		'\uFF0D': // fullwidth hyphen-minus
		return true
	}
	return false
}
