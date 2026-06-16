package needleman_wunsch

// TimedChar is a character with associated audio timestamps and alignment error.
type TimedChar struct {
	Char  string
	Begin float64
	End   float64
	Error float64
}

// InterpolateTimestamps fills gaps (nil entries) in matched with linearly
// interpolated timestamps and Error = 1.0. Non-nil entries are copied as-is.
func InterpolateTimestamps(matched []*TimedChar) []TimedChar {
	n := len(matched)
	result := make([]TimedChar, n)
	for i, c := range matched {
		if c != nil {
			result[i] = *c
		}
	}

	for i := 0; i < n; {
		if matched[i] != nil {
			i++
			continue
		}
		// Find the end of the gap.
		j := i
		for j < n && matched[j] == nil {
			j++
		}
		// Determine the time interval to spread across the gap.
		var prevEnd float64
		if i > 0 {
			prevEnd = result[i-1].End
		} else if j < n {
			prevEnd = result[j].Begin
		}
		var nextBegin float64
		if j < n {
			nextBegin = result[j].Begin
		} else {
			nextBegin = prevEnd
		}
		gapLen := j - i
		step := 0.0
		if gapLen > 0 {
			step = (nextBegin - prevEnd) / float64(gapLen)
		}
		for k := 0; k < gapLen; k++ {
			result[i+k] = TimedChar{
				Begin: prevEnd + float64(k)*step,
				End:   prevEnd + float64(k+1)*step,
				Error: 1.0,
			}
		}
		i = j
	}
	return result
}
