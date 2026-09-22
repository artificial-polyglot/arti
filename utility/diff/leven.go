package diff

// CharLevenshtein aligns refText (reference) against asrText (hypothesis) at the
// character (rune) level using a classic Levenshtein dynamic-programming
// alignment where substitution, insertion and deletion each cost 1.
//
// Unlike the diff-match-patch path, substitution is a first-class operation, so
// a changed character comes back as a single OpReplace (Char=ref rune,
// Replace=asr rune) rather than a delete+insert pair — and because it works on
// atomic runes it never splits a multibyte rune or exploits incidental shared
// characters between unrelated words.
//
// The result is a minimal-cost alignment; when several are minimal, ties are
// broken in a fixed order (match/substitute, then delete, then insert), which
// left-aligns edits consistently.
func CharLevenshtein(refText string, asrText string) []CDiff {
	ref := []rune(refText)
	hyp := []rune(asrText)
	n := len(ref)
	m := len(hyp)

	// cost[i][j] = edit distance between ref[:i] and hyp[:j].
	cost := make([][]int, n+1)
	for i := range cost {
		cost[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		cost[i][0] = i // i deletions
	}
	for j := 1; j <= m; j++ {
		cost[0][j] = j // j insertions
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			subCost := 1
			if ref[i-1] == hyp[j-1] {
				subCost = 0
			}
			cost[i][j] = min3(
				cost[i-1][j-1]+subCost, // substitute or match
				cost[i-1][j]+1,         // delete ref[i-1]
				cost[i][j-1]+1,         // insert hyp[j-1]
			)
		}
	}

	// Backtrace from (n,m) to (0,0), preferring the diagonal (match/sub) on
	// ties, then delete, then insert. Build the ops in reverse, then flip.
	var rev []CDiff
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 {
			subCost := 1
			if ref[i-1] == hyp[j-1] {
				subCost = 0
			}
			if cost[i][j] == cost[i-1][j-1]+subCost {
				if subCost == 0 {
					rev = append(rev, CDiff{Type: OpEqual, Char: ref[i-1]})
				} else {
					rev = append(rev, CDiff{Type: OpReplace, Char: ref[i-1], Replace: hyp[j-1]})
				}
				i--
				j--
				continue
			}
		}
		if i > 0 && cost[i][j] == cost[i-1][j]+1 {
			rev = append(rev, CDiff{Type: OpDelete, Char: ref[i-1]})
			i--
			continue
		}
		// remaining case: insertion
		rev = append(rev, CDiff{Type: OpInsert, Char: hyp[j-1]})
		j--
	}

	// reverse
	result := make([]CDiff, len(rev))
	for k := range rev {
		result[k] = rev[len(rev)-1-k]
	}
	return result
}

func min3(a, b, c int) int {
	return min(a, min(b, c))
}

// Counts holds the S/D/I/hits breakdown of an alignment.
type Counts struct {
	Hits          int // equal
	Substitutions int // replace
	Deletions     int // delete (in reference, not in hypothesis)
	Insertions    int // insert (in hypothesis, not in reference)
	RefLen        int // reference tokens = Hits + Substitutions + Deletions
}

// Count tallies a []CDiff into S/D/I/hits.
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
