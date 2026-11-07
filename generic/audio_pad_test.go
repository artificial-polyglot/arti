package generic

import "testing"

func TestPadBeginEnd(t *testing.T) {
	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"full padding before", PadBegin(5, 4, true), 4.8},
		{"half the gap before", PadBegin(5, 4.9, true), 4.95},
		{"start of file", PadBegin(0.1, 0, false), 0},
		{"full padding after", PadEnd(5, 6, true), 5.2},
		{"half the gap after", PadEnd(5, 5.1, true), 5.05},
		{"end of file", PadEnd(5, 0, false), 5.2},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}
