package filters

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/generic"
)

// boundaryFixture is ROM 12:18-21 of N1SKNSEC run 00003, the only boundary errors in the dataset.
// The boundary of ROM 12:19-20 is too early: the end of nilon is heard at the start of 12:20.
// The boundary of ROM 12:20-21 is too late: the start of naꞌ is heard at the end of 12:20.
type boundaryFixture struct {
	SecPerChar float64
	Next       map[int64]int64
	Verses     []generic.Verse2
}

func readBoundaryFixture(t *testing.T) boundaryFixture {
	var fx boundaryFixture
	b, err := os.ReadFile("testdata/n1sknsec_00003_boundaries.json_test")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	return fx
}

func TestFixBoundariesN1SKNSEC(t *testing.T) {
	fx := readBoundaryFixture(t)
	before := make([]string, len(fx.Verses))
	for i, vs := range fx.Verses {
		before[i] = vs.ASRText
	}
	firstChar := fx.Verses[3].Words[0].Chars[0]
	result, errors := FixBoundaries(fx.Verses, fx.Next, fx.SecPerChar)
	if len(errors) != 2 {
		t.Fatalf("expected 2 boundary errors, got %d: %v", len(errors), errors)
	}
	if e := errors[0]; !e.Later || e.Text != "ilon" || refString(e.LineRef) != "ROM 12:19" {
		t.Errorf("first error: %s", e.Reason())
	}
	if e := errors[1]; e.Later || e.Text != "na" || refString(e.LineRef) != "ROM 12:20" {
		t.Errorf("second error: %s", e.Reason())
	}
	if !strings.HasSuffix(result[1].ASRText, "sog kolaatan ilon") {
		t.Errorf("ROM 12:19 ASR: %q", result[1].ASRText)
	}
	if !strings.HasPrefix(result[2].ASRText, "unut niyu") || !strings.HasSuffix(result[2].ASRText, "diyalom kitab") {
		t.Errorf("ROM 12:20 ASR: %q", result[2].ASRText)
	}
	if !strings.HasPrefix(result[3].ASRText, "na amu meleg") {
		t.Errorf("ROM 12:21 ASR: %q", result[3].ASRText)
	}
	for i := 0; i+1 < len(result); i++ {
		if result[i].PlayEndTS != result[i+1].PlayBeginTS {
			t.Errorf("play times of %s and the next verse do not meet: %.3f %.3f", refString(result[i].LineRef),
				result[i].PlayEndTS, result[i+1].PlayBeginTS)
		}
	}
	shift := 4 * fx.SecPerChar
	if math.Abs(result[1].PlayEndTS-(fx.Verses[1].PlayEndTS+shift)) > 1e-9 {
		t.Errorf("ROM 12:19 play end %.3f, expected %.3f", result[1].PlayEndTS, fx.Verses[1].PlayEndTS+shift)
	}
	nilon := result[1].Words[len(result[1].Words)-1]
	if nilon.Text != "nilon" || math.Abs(nilon.EndTS-result[1].PlayEndTS) > 1e-9 {
		t.Errorf("nilon should end at the new boundary: %s %.3f", nilon.Text, nilon.EndTS)
	}
	if naa := result[3].Words[0]; math.Abs(naa.BeginTS-result[3].PlayBeginTS) > 1e-9 {
		t.Errorf("naꞌ should begin at the new boundary: %.3f %.3f", naa.BeginTS, result[3].PlayBeginTS)
	}
	for i, vs := range fx.Verses {
		if vs.ASRText != before[i] {
			t.Errorf("input verse %d was changed", i)
		}
	}
	if fx.Verses[3].Words[0].Chars[0] != firstChar {
		t.Error("input chars were changed")
	}
}

// pair makes two verses that follow each other, read at 0.1 seconds per char.
func pair(text1, asr1, text2, asr2 string) []generic.Verse2 {
	var verses []generic.Verse2
	ts := 0.0
	for i, text := range []string{text1, text2} {
		vs := generic.Verse2{ScriptId: int64(i + 1), PlayBeginTS: ts}
		for _, wd := range strings.Fields(text) {
			word := generic.Word2{Text: wd, BeginTS: ts}
			for _, r := range wd {
				word.Chars = append(word.Chars, generic.Char2{Char: r, BeginTS: ts, EndTS: ts + 0.1})
				ts += 0.1
			}
			word.EndTS = ts
			vs.Words = append(vs.Words, word)
			ts += 0.1
		}
		vs.PlayEndTS = ts
		verses = append(verses, vs)
	}
	verses[0].ASRText, verses[1].ASRText = asr1, asr2
	return verses
}

func TestFixBoundariesSynthetic(t *testing.T) {
	next := map[int64]int64{1: 2}
	cases := []struct {
		name               string
		text1, asr1        string
		text2, asr2        string
		want1, want2, text string
	}{
		{"too early, whole word", "aaaa bbbb cccc", "aaaa bbbb", "dddd eeee", "cccc dddd eeee",
			"aaaa bbbb cccc", "dddd eeee", "cccc"},
		{"too early, joined to next word", "aaaa bbbb cccc", "aaaa bbbb c", "dddd eeee", "cccdddd eeee",
			"aaaa bbbb cccc", "dddd eeee", "ccc"},
		{"too late", "aaaa bbbb cccc", "aaaa bbbb cccc dd", "dddd eeee", "dd eeee",
			"aaaa bbbb cccc", "dddd eeee", "dd"},
		{"too late, joined to last word", "aaaa bbbb cccc", "aaaa bbbb ccccdddd", "dddd eeee", "eeee",
			"aaaa bbbb cccc", "dddd eeee", "dddd"},
		{"repeat is not moved", "aaaa bbbb cccc", "aaaa bbbb cccc", "dddd eeee", "cccc dddd eeee",
			"aaaa bbbb cccc", "cccc dddd eeee", ""},
		{"omission is not moved", "aaaa bbbb cccc", "aaaa bbbb", "dddd eeee", "dddd eeee",
			"aaaa bbbb", "dddd eeee", ""},
		{"one letter is not moved", "aaaa bbbb cccc", "aaaa bbbb ccc", "dddd eeee", "cdddd eeee",
			"aaaa bbbb ccc", "cdddd eeee", ""},
	}
	for _, c := range cases {
		verses := pair(c.text1, c.asr1, c.text2, c.asr2)
		result, errors := FixBoundaries(verses, next, 0.1)
		if result[0].ASRText != c.want1 || result[1].ASRText != c.want2 {
			t.Errorf("%s: got %q | %q", c.name, result[0].ASRText, result[1].ASRText)
		}
		if c.text == "" {
			if len(errors) != 0 {
				t.Errorf("%s: expected no correction, got %s", c.name, errors[0].Reason())
			}
			continue
		}
		if len(errors) != 1 || errors[0].Text != c.text {
			t.Errorf("%s: expected [%s] moved, got %v", c.name, c.text, errors)
			continue
		}
		e := errors[0]
		shift := float64(len([]rune(strings.ReplaceAll(c.text, " ", "")))) * 0.1
		if !e.Later {
			shift = -shift
		}
		if math.Abs(e.NewBoundaryTS-(e.OldBoundaryTS+shift)) > 1e-9 ||
			result[0].PlayEndTS != e.NewBoundaryTS || result[1].PlayBeginTS != e.NewBoundaryTS {
			t.Errorf("%s: boundary %.2f -> %.2f, play %.2f %.2f", c.name, e.OldBoundaryTS, e.NewBoundaryTS,
				result[0].PlayEndTS, result[1].PlayBeginTS)
		}
	}
}

func TestFixBoundariesNotNeighbours(t *testing.T) {
	verses := pair("aaaa bbbb cccc", "aaaa bbbb", "dddd eeee", "cccc dddd eeee")
	for _, next := range []map[int64]int64{{}, {1: 3}} {
		result, errors := FixBoundaries(verses, next, 0.1)
		if len(errors) != 0 || result[1].ASRText != "cccc dddd eeee" {
			t.Errorf("verses that are not neighbours were corrected: %v", errors)
		}
	}
}

func TestOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"nilon", "ilonunut", 4},
		{"abc", "xyz", 0},
		{"abab", "abab", 4},
		{"", "abc", 0},
		{"na", "naꞌ", 2},
	}
	for _, c := range cases {
		if got := overlap([]rune(c.a), []rune(c.b)); got != c.want {
			t.Errorf("overlap(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
