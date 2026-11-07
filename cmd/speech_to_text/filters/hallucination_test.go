package filters

import (
	"strconv"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/generic"
)

// TestFilterN1SKNSEC checks the decisions on every whole-word insert in N1SKNSEC run 00003.
// ROM 3:26 is the verse read twice.  In ACT 23:31 the reader appears to restart a phrase, and in
// ACT 26:22 to read a phrase in another place.  The replaced inserts are words of the text read in
// another order, or misspelled words that the alignment matched to another copy of a word nearby.
// The other inserts are judged hallucinations.
func TestFilterN1SKNSEC(t *testing.T) {
	fx := readFixture(t)
	real := map[string]bool{"ROM 3:26": true, "ACT 23:31": true, "ACT 26:22": true}
	replaced := map[string]bool{"MAT 6:16": true, "LUK 13:16": true, "ACT 15:17": true, "EPH 1:2": true,
		"EPH 5:27": true, "2PE 1:20": true, "REV 8:10": true}
	result, inserts := Filter(fx.Verses, fx.Clips, fx.Stats)
	if len(inserts) != 14 {
		t.Errorf("expected 14 whole-word inserts, got %d", len(inserts))
	}
	for _, in := range inserts {
		ref := in.LineRef.BookId + " " + strconv.Itoa(in.LineRef.ChapterNum) + ":" + in.LineRef.VerseStr
		if in.Kept != (real[ref] || replaced[ref]) || (in.Replaces != "") != replaced[ref] {
			t.Errorf("%s +[%s]: kept %v, %s", ref, in.Text, in.Kept, in.Reason())
		}
	}
	for i, vs := range result {
		before, after := len(strings.Fields(fx.Verses[i].ASRText)), len(strings.Fields(vs.ASRText))
		ref := vs.LineRef.BookId + " " + strconv.Itoa(vs.LineRef.ChapterNum) + ":" + vs.LineRef.VerseStr
		keep := real[ref] || replaced[ref]
		if keep && after != before || !keep && after >= before {
			t.Errorf("%s: transcript went from %d to %d words", ref, before, after)
		}
	}
}

func TestReplacedWord(t *testing.T) {
	cases := []struct {
		name, text, asr, want string
	}{
		{"reordered", "aaaa pppp bbbb cccc", "aaaa bbbb pppp cccc", "pppp"},
		{"misspelled, matched to another copy", "tuhan og gommaꞌ taꞌ bug panghuꞌ ta isa al-masih", // EPH 1:2
			"tuhan og gommaꞌ ta bu panghuꞌ isa al-masih", "bug"},
		{"hallucination", "aaaa bbbb cccc dddd", "aaaa bbbb xyzw cccc dddd", ""},
		{"unlike missing word", "aaaa bbbb cccc dddd", "aaaa xyzw cccc dddd", ""},
		{"too far away", "pppp aaaa bbbb cccc dddd eeee ffff", "aaaa bbbb cccc dddd eeee ffff pppp", ""},
	}
	for _, c := range cases {
		text, asr := strings.Fields(c.text), strings.Fields(c.asr)
		runs, matched := wordInserts(text, asr)
		got := ""
		for _, run := range runs {
			if r := replacedWord(asr[run.asrBegin:run.asrEnd], text, matched, run.textIndex); r != "" {
				got = r
			}
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// verse makes a verse read at exactly the stats' speed, with the words of text and a transcript.
func verse(text string, asr string, stats Stats, gapBefore map[int]float64) (generic.Verse2, Clip) {
	vs := generic.Verse2{ScriptId: 1, ASRText: asr}
	ts := 1.0
	for i, wd := range strings.Fields(text) {
		ts += gapBefore[i]
		if i > 0 {
			ts += stats.WordGapSec
		}
		end := ts + float64(len(wd))*stats.SecPerChar
		vs.Words = append(vs.Words, generic.Word2{Text: wd, BeginTS: ts, EndTS: end})
		ts = end
	}
	return vs, Clip{BeginTS: 1.0 - generic.AudioPadSec, EndTS: ts + generic.AudioPadSec}
}

func TestFilterSynthetic(t *testing.T) {
	stats := Stats{SecPerChar: 0.07, WordGapSec: 0.08, NearLimitSec: 1.0, VerseLimitSec: 1.6}
	text := "aaaa bbbb cccc dddd eeee ffff gggg hhhh"
	cases := []struct {
		name string
		asr  string
		gaps map[int]float64
		want string
		kept int
	}{
		{"no room for the insert", "aaaa bbbb cccc xxxxxx dddd eeee ffff gggg hhhh", nil,
			"aaaa bbbb cccc dddd eeee ffff gggg hhhh", 0},
		{"a gap where the insert is", "aaaa bbbb cccc xxxxxxxxxxxx dddd eeee ffff gggg hhhh", map[int]float64{3: 1.2},
			"aaaa bbbb cccc xxxxxxxxxxxx dddd eeee ffff gggg hhhh", 1},
		{"a repeat absorbed elsewhere in the verse", "aaaa bbbb cccc dddd eeee ffff gggg hhhh aaaa bbbb cccc dddd eeee ffff",
			map[int]float64{2: 1.5, 4: 1.5}, "aaaa bbbb cccc dddd eeee ffff gggg hhhh aaaa bbbb cccc dddd eeee ffff", 1},
		{"short inserts are not judged", "aaaa bbbb cccc xx dddd eeee ffff gggg hhhh", nil,
			"aaaa bbbb cccc xx dddd eeee ffff gggg hhhh", 0},
		{"a replacement is not an insert", "aaaa bbbb cccc yyyy zzzz eeee ffff gggg hhhh", nil,
			"aaaa bbbb cccc yyyy zzzz eeee ffff gggg hhhh", 0},
		{"an insert at the start with no room", "xxxxxx aaaa bbbb cccc dddd eeee ffff gggg hhhh", nil,
			"aaaa bbbb cccc dddd eeee ffff gggg hhhh", 0},
	}
	for _, c := range cases {
		vs, clip := verse(text, c.asr, stats, c.gaps)
		result, inserts := Filter([]generic.Verse2{vs}, map[int64]Clip{1: clip}, stats)
		if result[0].ASRText != c.want {
			t.Errorf("%s: got %q", c.name, result[0].ASRText)
		}
		kept := 0
		for _, in := range inserts {
			if in.Kept {
				kept++
			}
		}
		if kept != c.kept {
			t.Errorf("%s: kept %d inserts, %+v", c.name, kept, inserts)
		}
	}
}

func TestWordInserts(t *testing.T) {
	text := strings.Fields("a b c d")
	got, _ := wordInserts(text, strings.Fields("x a b y z c q d w"))
	want := []insertRun{{0, 0, 1}, {2, 3, 5}, {3, 6, 7}, {4, 8, 9}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("run %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPercentile(t *testing.T) {
	xs := []float64{5, 1, 4, 2, 3, 6, 7, 8, 9, 10}
	if percentile(xs, 90) != 9 || percentile(xs, 50) != 5 || percentile(nil, 90) != 0 {
		t.Error(percentile(xs, 90), percentile(xs, 50))
	}
}
