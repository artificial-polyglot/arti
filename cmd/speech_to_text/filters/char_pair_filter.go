package filters

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/utility/diff"
	"golang.org/x/text/unicode/norm"
)

// A filter for chars that ASR commonly writes as another char, copied from proofing_rpt's
// CharFilter1.  It counts, over the whole dataset, how often each char of the text is replaced by
// each other char in the transcripts.  A pair that replaces more than minPairRate percent of the
// occurrences of its char is treated as a spelling difference, not an error.  Where a replaced part
// of a transcript differs from the text only by such pairs, the transcript is given the text's chars.
//
// Three kinds of difference are undone however often they occur:
//   - diacritics: a letter written with or without a diacritic, such as é for e, or ড় for ড (the
//     nukta).  Two chars are the same letter when Unicode decomposes them (NFD) to the same base
//     and diacritics only.  An inserted or deleted diacritic is also undone.
//   - equivalent vowels: a vowel in EquivalentVowels written as the other vowel of its pair, such
//     as the long and short i of Bengali script.
//   - ignorable chars: a single inserted or deleted char that is ignorable, the Bengali virama.
// The verses' words, chars and times are not changed: only ASRText.

const MinPairCount = 3 // a pair seen fewer times is too rare to trust its rate

// DefaultPairRate is the minPairRate of CharFilter1.  It is the percent of a char's occurrences that
// must be replaced by the pair.  A script with letters that often sound alike may need less: in
// N2CCPBBS (Chakma), 2.0 finds 8 pairs between 2.0 and 4.4 percent, and 5.0 finds none.
const DefaultPairRate = 5.0

// The rules, other than common pairs, that can explain a difference.
const (
	RuleDiacritic = "diacritic"
	RuleVowel     = "equivalent vowel"
	RuleIgnorable = "ignorable char"
)

// EquivalentVowels are vowels that are treated as the same.  They are pairs of separate chars to
// Unicode, so they must be listed.  Each pair should be confirmed by someone who knows the script.
var EquivalentVowels = [][2]rune{
	{0x0987, 0x0988}, // Bengali letter I, II
	{0x0989, 0x098A}, // Bengali letter U, UU
	{0x09BF, 0x09C0}, // Bengali vowel sign I, II
	{0x09C1, 0x09C2}, // Bengali vowel sign U, UU
}

// diacritics are the combining marks treated as diacritics.  The marks that Unicode decomposes
// from letters also include vowel signs, subjoined consonants and voicing marks, which change the
// sound, so the diacritics are listed instead.
var diacritics = func() map[rune]bool {
	result := map[rune]bool{
		0x0653: true, 0x0654: true, 0x0655: true, // Arabic maddah, hamza above and below
		0x093C: true, 0x09BC: true, 0x0A3C: true, 0x0B3C: true, 0x110BA: true, // nuktas
	}
	for r := rune(0x0300); r <= 0x036F; r++ { // Combining Diacritical Marks
		result[r] = true
	}
	return result
}()

// CharPair is a char of the text and the char that ASR commonly writes for it.
type CharPair struct {
	From      rune
	To        rune
	Count     int     // times From was replaced by To
	CharCount int     // times From occurs in the text
	Rate      float64 // percent of the occurrences of From that were replaced by To
}

func (p CharPair) String() string {
	return string(p.From) + "→" + string(p.To) + " " + strconv.FormatFloat(p.Rate, 'f', 1, 64) + "%"
}

// CharPairChange is one part of a transcript that was given the text's chars.
type CharPairChange struct {
	ScriptId int64
	LineRef  generic.VerseRef
	Text     string     // the text's chars, now in the transcript
	ASR      string     // the chars the transcript had
	Pairs    []CharPair // the common pairs that explain the difference
	Rules    []string   // the other rules that explain it
}

func (c CharPairChange) Reason() string {
	reasons := slices.Clone(c.Rules)
	for _, p := range c.Pairs {
		reasons = append(reasons, "common pair "+p.String())
	}
	return "[" + c.ASR + "] -> [" + c.Text + "]: " + strings.Join(reasons, ", ")
}

// FilterCharPairs returns a copy of verses with the common char pairs in each ASRText changed to
// the text's chars, and every change made.  minPairRate is the percent of a char's occurrences that
// a pair must replace, DefaultPairRate unless the dataset needs another.  verses must have ASR transcripts, as
// proofing_rpt.AddASRTranscript returns them.
func FilterCharPairs(verses []generic.Verse2, minPairRate float64) ([]generic.Verse2, []CharPairChange) {
	pairs := FindCharPairs(verses, minPairRate)
	result := make([]generic.Verse2, len(verses))
	var changes []CharPairChange
	for i, vs := range verses {
		result[i] = vs
		if len(vs.Words) == 0 {
			continue
		}
		asr, verseChanges := applyCharPairs(vs, pairs)
		if len(verseChanges) > 0 {
			result[i].ASRText = asr
			changes = append(changes, verseChanges...)
		}
	}
	return result, changes
}

// FindCharPairs returns the common pairs of the dataset, by the char of the text and the char of ASR.
func FindCharPairs(verses []generic.Verse2, minPairRate float64) map[rune]map[rune]CharPair {
	charCounts := make(map[rune]int)
	pairCounts := make(map[diff.CDiff]int)
	for _, vs := range verses {
		text := vs.Text()
		for _, r := range text {
			charCounts[r]++
		}
		for _, d := range diff.CharLevenshtein(text, vs.ASRText) {
			if d.Type == diff.OpReplace {
				pairCounts[d]++
			}
		}
	}
	result := make(map[rune]map[rune]CharPair)
	for d, count := range pairCounts {
		charCount := charCounts[d.Char]
		if count < MinPairCount || charCount == 0 {
			continue
		}
		if unicode.IsDigit(d.Char) || unicode.IsDigit(d.Replace) {
			continue // a number should be compared as a word, not by its chars
		}
		pair := CharPair{From: d.Char, To: d.Replace, Count: count, CharCount: charCount,
			Rate: 100.0 * float64(count) / float64(charCount)}
		if pair.Rate <= minPairRate {
			continue
		}
		if result[d.Char] == nil {
			result[d.Char] = make(map[rune]CharPair)
		}
		result[d.Char][d.Replace] = pair
	}
	return result
}

// applyCharPairs returns the verse's transcript with every part explained by common pairs or the
// rules given the text's chars.
func applyCharPairs(vs generic.Verse2, pairs map[rune]map[rune]CharPair) (string, []CharPairChange) {
	var asr strings.Builder
	var changes []CharPairChange
	for _, df := range diff.DiffReplace(vs.Text(), vs.ASRText) {
		change := CharPairChange{ScriptId: vs.ScriptId, LineRef: vs.LineRef, Text: df.Text}
		switch df.Type {
		case diff.OpEqual:
			asr.WriteString(df.Text)
		case diff.OpReplace:
			used, rules, ok := explain(df, pairs)
			if ok {
				asr.WriteString(df.Text)
				change.ASR, change.Pairs, change.Rules = df.Replace, used, rules
				changes = append(changes, change)
			} else {
				asr.WriteString(df.Replace)
			}
		case diff.OpInsert:
			if rule := aloneRule(df.Text); rule != "" {
				change.Text, change.ASR, change.Rules = "", df.Text, []string{rule}
				changes = append(changes, change)
			} else {
				asr.WriteString(df.Text)
			}
		case diff.OpDelete:
			if rule := aloneRule(df.Text); rule != "" {
				asr.WriteString(df.Text)
				change.Rules = []string{rule}
				changes = append(changes, change)
			}
		}
	}
	return asr.String(), changes
}

// explain returns the common pairs and the rules that explain every difference between the text
// and the transcript of a replaced part, and false if any difference is not explained.
func explain(df diff.Diff, pairs map[rune]map[rune]CharPair) ([]CharPair, []string, bool) {
	var used []CharPair
	var rules []string
	addRule := func(rule string) {
		if !slices.Contains(rules, rule) {
			rules = append(rules, rule)
		}
	}
	for _, ch := range diff.CharLevenshtein(df.Text, df.Replace) {
		switch ch.Type {
		case diff.OpEqual:
			continue
		case diff.OpReplace:
			if baseLetter(ch.Char) == baseLetter(ch.Replace) {
				addRule(RuleDiacritic)
			} else if equivalentVowels(ch.Char, ch.Replace) {
				addRule(RuleVowel)
			} else if pair, ok := pairs[ch.Char][ch.Replace]; ok {
				if !slices.Contains(used, pair) {
					used = append(used, pair)
				}
			} else {
				return nil, nil, false
			}
		default:
			rule := charRule(ch.Char)
			if rule == "" {
				return nil, nil, false
			}
			addRule(rule)
		}
	}
	return used, rules, true
}

// aloneRule returns the rule that explains text inserted or deleted on its own: a single char that
// is ignorable or a diacritic.  It returns "" for any other text.
func aloneRule(text string) string {
	if utf8.RuneCountInString(text) != 1 {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(text)
	return charRule(r)
}

// charRule returns the rule that explains a char inserted or deleted, or "" if none does.
func charRule(r rune) string {
	if isIgnorable(r) {
		return RuleIgnorable
	}
	if diacritics[r] {
		return RuleDiacritic
	}
	return ""
}

// baseLetter returns the letter a char decomposes to when the rest of it is diacritics, and
// otherwise the char itself.
func baseLetter(r rune) rune {
	runes := []rune(norm.NFD.String(string(r)))
	for _, m := range runes[1:] {
		if !diacritics[m] {
			return r
		}
	}
	return runes[0]
}

func equivalentVowels(a rune, b rune) bool {
	for _, pair := range EquivalentVowels {
		if pair == [2]rune{a, b} || pair == [2]rune{b, a} {
			return true
		}
	}
	return false
}

func isIgnorable(r rune) bool {
	return r == 0x09CD // BENGALI SIGN VIRAMA
}
