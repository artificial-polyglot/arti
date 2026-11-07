package filters

import (
	"slices"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/utility/diff"
)

// textVerse makes a verse of text and its transcript.
func textVerse(id int64, text string, asr string) generic.Verse2 {
	vs := generic.Verse2{ScriptId: id, ASRText: asr}
	for _, wd := range strings.Fields(text) {
		vs.Words = append(vs.Words, generic.Word2{Text: wd})
	}
	return vs
}

// pairDataset has o written as u in 5 of its 23 o's, and d and e each written another way once.
func pairDataset() []generic.Verse2 {
	return []generic.Verse2{
		textVerse(1, "tolo moto bobo", "tulo muto bobo"),
		textVerse(2, "koto loko", "kuto luko"),
		textVerse(3, "oooo oooo", "oooo oooo"),
		textVerse(4, "sopo nemo", "sopo nimo"),
		textVerse(5, "kopa dego", "kupa xego"),
	}
}

func TestFindCharPairs(t *testing.T) {
	pairs := FindCharPairs(pairDataset(), DefaultPairRate)
	ou, ok := pairs['o']['u']
	if !ok || ou.Count != 5 || ou.CharCount != 23 {
		t.Fatalf("expected o→u 5 of 23, got %+v (all: %v)", ou, pairs)
	}
	if _, ok := pairs['e']['i']; ok {
		t.Error("e→i is seen once, which is too rare to be a pair")
	}
	if pairs = FindCharPairs(pairDataset(), 25.0); len(pairs) != 0 {
		t.Errorf("o→u at 21.7%% should not pass a rate of 25%%: %v", pairs)
	}
}

func TestFilterCharPairs(t *testing.T) {
	verses := pairDataset()
	result, changes := FilterCharPairs(verses, DefaultPairRate)
	want := []string{"tolo moto bobo", "koto loko", "oooo oooo", "sopo nimo", "kopa xego"}
	for i, vs := range result {
		if vs.ASRText != want[i] {
			t.Errorf("verse %d: got %q, want %q", vs.ScriptId, vs.ASRText, want[i])
		}
	}
	if verses[0].ASRText != "tulo muto bobo" {
		t.Error("the input verses were changed")
	}
	for _, c := range changes {
		if len(c.Pairs) != 1 || c.Pairs[0].From != 'o' || c.Pairs[0].To != 'u' {
			t.Errorf("verse %d: %s", c.ScriptId, c.Reason())
		}
	}
	if len(changes) != 5 {
		t.Errorf("expected 5 changes, got %d", len(changes))
	}
}

func TestFilterCharPairsIgnorable(t *testing.T) {
	virama := string(rune(0x09CD))
	verses := []generic.Verse2{
		textVerse(1, "কম"+virama+" বল", "কম বল"),
		textVerse(2, "কম বল", "কম বল"+virama),
	}
	result, changes := FilterCharPairs(verses, DefaultPairRate)
	if result[0].ASRText != "কম"+virama+" বল" || result[1].ASRText != "কম বল" {
		t.Errorf("got %q and %q", result[0].ASRText, result[1].ASRText)
	}
	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}
}

// TestSameAsCharFilter1 checks that the filter undoes the same differences as proofing_rpt's
// CharFilter1, on the N1SKNSEC verses of the hallucination fixture.
func TestSameAsCharFilter1(t *testing.T) {
	verses := append(readFixture(t).Verses, pairDataset()...)
	charFilter := proofing_rpt.NewCharFilter1(verses)
	result, _ := FilterCharPairs(verses, DefaultPairRate)
	for i, vs := range verses {
		diffs := charFilter.DiffFilter(diff.DiffReplace(vs.Text(), vs.ASRText))
		var want strings.Builder
		for _, df := range diffs {
			switch df.Type {
			case diff.OpEqual:
				want.WriteString(df.Text)
			case diff.OpReplace:
				want.WriteString(df.Replace)
			case diff.OpInsert:
				want.WriteString(df.Text)
			}
		}
		if len(vs.Words) > 0 && result[i].ASRText != want.String() {
			t.Errorf("verse %d: got %q, CharFilter1 gives %q", vs.ScriptId, result[i].ASRText, want.String())
		}
	}
}

// TestRulesAtAnyFrequency checks the rules that apply even when a difference occurs once.
func TestRulesAtAnyFrequency(t *testing.T) {
	acute, nukta := string(rune(0x0301)), string(rune(0x09BC))
	cases := []struct {
		name, text, asr, want, rule string
	}{
		{"accented letter", "café noir", "cafe noir", "café noir", RuleDiacritic},
		{"different accent", "café noir", "cafè noir", "café noir", RuleDiacritic},
		{"decomposed accent", "cafe" + acute + " noir", "cafe noir", "cafe" + acute + " noir", RuleDiacritic},
		{"nukta", "পড" + nukta + "া বই", "পডা বই", "পড" + nukta + "া বই", RuleDiacritic},
		{"composed nukta", "প" + string(rune(0x09DC)) + "া বই", "পডা বই", "প" + string(rune(0x09DC)) + "া বই", RuleDiacritic},
		{"long and short i", "নদী বই", "নদি বই", "নদী বই", RuleVowel},
		{"long and short u", "ভূত বই", "ভুত বই", "ভূত বই", RuleVowel},
		{"vowel sign is not a diacritic", "কুল বই", "কল বই", "কল বই", ""},
		{"candrabindu is not a diacritic", "চাঁদ বই", "চাদ বই", "চাদ বই", ""},
		{"other letters", "café noir", "cafa noir", "cafa noir", ""},
	}
	for _, c := range cases {
		result, changes := FilterCharPairs([]generic.Verse2{textVerse(1, c.text, c.asr)}, DefaultPairRate)
		if result[0].ASRText != c.want {
			t.Errorf("%s: got %q, want %q", c.name, result[0].ASRText, c.want)
		}
		if c.rule == "" {
			if len(changes) != 0 {
				t.Errorf("%s: expected no change, got %s", c.name, changes[0].Reason())
			}
		} else if len(changes) != 1 || !slices.Equal(changes[0].Rules, []string{c.rule}) {
			t.Errorf("%s: expected one change by %s, got %v", c.name, c.rule, changes)
		}
	}
}

func TestBaseLetter(t *testing.T) {
	cases := []struct{ r, want rune }{
		{'é', 'e'}, {'ñ', 'n'}, {'a', 'a'},
		{0x09DC, 0x09A1}, // Bengali RRA is DDA and a nukta
		{0x0995, 0x0995}, // Bengali KA
		{0x1026, 0x1026}, // Myanmar UU decomposes to a vowel sign, which is not a diacritic
	}
	for _, c := range cases {
		if got := baseLetter(c.r); got != c.want {
			t.Errorf("baseLetter(%U) = %U, want %U", c.r, got, c.want)
		}
	}
}
