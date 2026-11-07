package filters

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/generic"
)

// missingFixture is from N1SKNSEC run 00003, after FixBoundaries: the 7 verses of the known errors
// in N1SKNSEC_Audio_Errors.csv, the 9 verses with a doubtful missing word, the 3 verses where
// Antipas is missing, and a chapter heading.  Each word keeps one char, with the mean qa_align score.
type missingFixture struct {
	MMSScores map[int64]float64
	Verses    []generic.Verse2
}

func readMissingFixture(t *testing.T) missingFixture {
	var fx missingFixture
	b, err := os.ReadFile("testdata/n1sknsec_00003_missing.json_test")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	return fx
}

func TestFindMissingWordsN1SKNSEC(t *testing.T) {
	fx := readMissingFixture(t)
	missing := FindMissingWords(fx.Verses, fx.MMSScores)
	found := make(map[string][]MissingWord) // a word can be in a verse more than once
	for _, m := range missing {
		key := refString(m.LineRef) + " " + strings.ToLower(m.Text)
		found[key] = append(found[key], m)
		if hasDigit(m.Text) {
			t.Errorf("%s: a number was reported: %s", refString(m.LineRef), m.Text)
		}
	}
	known := map[string]string{
		"MAT 16:10": "bu og",
		"MAT 21:33": "inongogoy og len pononggiꞌ kini ongon ok solokotow nog gapuꞌ lupaꞌ nong mimula anggul",
		"MRK 2:22":  "bogu",
		"LUK 14:11": "nong",
		"2CO 7:12":  "bagun motampal diyon diniyu",
		"1TH 2:9":   "mga pigilugan",
		"1PE 2:4":   "lolayun",
		"MAT 14:1":  "antipas",
		"MRK 6:14":  "antipas",
		"LUK 9:7":   "antipas",
	}
	for ref, words := range known {
		for _, wd := range strings.Fields(words) {
			ms := found[ref+" "+wd]
			if len(ms) == 0 {
				t.Errorf("%s %s: not reported as missing", ref, wd)
			}
			for _, m := range ms {
				if !m.Confirmed {
					t.Errorf("%s %s: %s", ref, wd, m.Reason())
				}
			}
		}
	}
	for _, key := range []string{"MAT 16:10 dakol", "MAT 21:33 poglogekan"} {
		if _, ok := found[key]; ok {
			t.Errorf("%s: a partly missing word was reported", key)
		}
	}
	doubtful := []string{"LUK 3:9 sopantun", "JHN 17:1 u", "ACT 13:33 sura", "ROM 1:1 sog", "ROM 12:20 bu",
		"2CO 1:2 ta", "EPH 1:2 ta", "1TH 3:11 ta", "2TI 1:1 ta"}
	count := 0
	for _, m := range missing {
		if !m.Confirmed {
			count++
		}
	}
	if count != len(doubtful) {
		t.Errorf("expected %d doubtful words, got %d", len(doubtful), count)
	}
	for _, key := range doubtful {
		doubt := false
		for _, m := range found[key] {
			doubt = doubt || !m.Confirmed
		}
		if !doubt {
			t.Errorf("%s: expected doubtful, got %+v", key, found[key])
		}
	}
}

func TestFindMissingWordsSynthetic(t *testing.T) {
	text := "aaaa bbbb cccc dddd 12"
	vs := generic.Verse2{ScriptId: 1}
	for i, wd := range strings.Fields(text) {
		vs.Words = append(vs.Words, generic.Word2{WordId: int64(i + 1), Text: wd,
			Chars: []generic.Char2{{FAScore: 0.0}}})
	}
	mms := map[int64]float64{1: 1.0, 2: 0.01, 3: 0.9, 4: 0.0, 5: 0.0}
	cases := []struct {
		name string
		asr  string
		want map[string]bool // missing word and whether it is confirmed
	}{
		{"nothing missing", "aaaa bbbb cccc dddd", map[string]bool{}},
		{"missing from audio", "aaaa cccc dddd", map[string]bool{"bbbb": true}},
		{"base model heard it", "aaaa bbbb dddd", map[string]bool{"cccc": false}},
		{"partly missing", "aaaa bb cccc dddd", map[string]bool{}},
		{"joined to next word", "aaaa bbbbcccc dddd", map[string]bool{}},
		{"two words", "aaaa dddd", map[string]bool{"bbbb": true, "cccc": false}},
	}
	for _, c := range cases {
		vs.ASRText = c.asr
		missing := FindMissingWords([]generic.Verse2{vs}, mms)
		got := make(map[string]bool)
		for _, m := range missing {
			got[m.Text] = m.Confirmed
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
			continue
		}
		for wd, conf := range c.want {
			if g, ok := got[wd]; !ok || g != conf {
				t.Errorf("%s: got %v, want %v", c.name, got, c.want)
			}
		}
	}
}
