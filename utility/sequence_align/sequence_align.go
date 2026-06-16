package sequence_align

import "github.com/sergi/go-diff/diffmatchpatch"

// Align aligns ref and query using Myers' diff algorithm and returns the
// alignment as a slice of [refIdx, queryIdx] pairs. A value of -1 means a gap.
func Align(ref, query []rune) [][2]int {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(ref), string(query), false)

	var alignment [][2]int
	ri, qi := 0, 0
	for _, d := range diffs {
		runes := []rune(d.Text)
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			for range runes {
				alignment = append(alignment, [2]int{ri, qi})
				ri++
				qi++
			}
		case diffmatchpatch.DiffDelete:
			for range runes {
				alignment = append(alignment, [2]int{ri, -1})
				ri++
			}
		case diffmatchpatch.DiffInsert:
			for range runes {
				alignment = append(alignment, [2]int{-1, qi})
				qi++
			}
		}
	}
	return alignment
}
