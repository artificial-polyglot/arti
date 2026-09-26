package safe

import "testing"

func TestSafeVerseNum(t *testing.T) {
	cases := map[string]int{
		"12":  12,
		"6a":  6,
		"3-4": 3,
		"":    0,
		"x":   0,
		"١٢":  12, // Arabic-Indic digits
		"३४":  34, // Devanagari digits
		"১৯b": 19, // Bengali digits, then a letter
	}
	for in, want := range cases {
		if got := SafeVerseNum(in); got != want {
			t.Errorf("SafeVerseNum(%q) = %d, want %d", in, got, want)
		}
	}
}
