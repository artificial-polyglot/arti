package needleman_wunsch

// NWAlign runs Needleman-Wunsch on ref and query and returns the alignment as
// a slice of [refIdx, queryIdx] pairs. A value of -1 means a gap at that side.
func NWAlign(ref, query []rune) [][2]int {
	const (
		match    = 2
		mismatch = -1
		gap      = -2
	)

	m := len(ref)
	n := len(query)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 0; i <= m; i++ {
		dp[i][0] = i * gap
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j * gap
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			sc := mismatch
			if ref[i-1] == query[j-1] {
				sc = match
			}
			dp[i][j] = max3(dp[i-1][j-1]+sc, dp[i-1][j]+gap, dp[i][j-1]+gap)
		}
	}

	// Traceback from (m, n) to (0, 0).
	var alignment [][2]int
	i, j := m, n
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && dp[i][j] == dp[i-1][j-1]+scoreOf(ref[i-1], query[j-1], match, mismatch):
			alignment = append(alignment, [2]int{i - 1, j - 1})
			i--
			j--
		case i > 0 && dp[i][j] == dp[i-1][j]+gap:
			alignment = append(alignment, [2]int{i - 1, -1})
			i--
		default:
			alignment = append(alignment, [2]int{-1, j - 1})
			j--
		}
	}

	// Reverse: traceback builds the alignment in reverse order.
	for l, r := 0, len(alignment)-1; l < r; l, r = l+1, r-1 {
		alignment[l], alignment[r] = alignment[r], alignment[l]
	}
	return alignment
}

func scoreOf(a, b rune, match, mismatch int) int {
	if a == b {
		return match
	}
	return mismatch
}

func max3(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}
