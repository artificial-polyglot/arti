package diff

import "unicode/utf8"

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
func (c Counts) ErrorRate() float64 {
	if c.RefLen == 0 {
		return 0
	}
	return float64(c.Substitutions+c.Deletions+c.Insertions) / float64(c.RefLen)
}

func (c Counts) ErrorCount() int {
	return c.Substitutions + c.Deletions + c.Insertions
}
