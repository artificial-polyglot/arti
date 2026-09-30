package proofing_rpt

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
)

func AddASRTranscript(conn db.DBAdapter, verses []generic.Verse2) *log.Status {
	align := NewAlignSilence(conn)
	for i := range verses {
		asrText, status := align.SelectTranscript(verses[i].ScriptId)
		if status != nil {
			return status
		}
		verses[i].ASRText = asrText
	}
	return nil
}

func CountCharOccurances(verses []generic.Verse2) map[rune]int {
	var countChars = make(map[rune]int)
	for _, vs := range verses {
		for _, r := range vs.Text() {
			countChars[r]++
		}
	}
	return countChars
}

func FindSimilarChars(verses []generic.Verse2) map[diff.CDiff]int {
	var countReplace = make(map[diff.CDiff]int)
	for _, vs := range verses {
		cdiff := diff.CharLevenshtein(vs.Text(), vs.ASRText)
		for _, d := range cdiff {
			if d.Type == diff.OpReplace {
				countReplace[d]++
			}
		}
	}
	return countReplace
}

type ReplaceCount struct {
	Diff       diff.CDiff
	Count      int     // times Char was replaced by Replace
	CharCount  int     // times Char occurs in the reference text
	Rate       float64 // % of Char occurrences replaced by Replace
	Percentile float64 // % of distinct pairs with a lower rate
	//CumShare   float64 // % of all replacements covered by this pair and all above it
}

func CreateReplaceCounts(charCounts map[rune]int, diffCounts map[diff.CDiff]int, minCount int) []ReplaceCount {
	var counts []ReplaceCount
	var total int
	for d, cnt := range diffCounts {
		total += 1
		charCnt := charCounts[d.Char]
		if cnt < minCount || charCnt == 0 {
			continue // too rare to trust the rate
		}
		if unicode.IsDigit(d.Char) || unicode.IsDigit(d.Replace) {
			continue // numeric substitution should be handled by word not char
		}
		counts = append(counts, ReplaceCount{
			Diff:      d,
			Count:     cnt,
			CharCount: charCnt,
			Rate:      100.0 * float64(cnt) / float64(charCnt),
		})
	}
	return counts
}

func RankReplaceCounts(ranked []ReplaceCount) {
	// Highest rate first; then higher count, then char, so map order doesn't change the output
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Rate != ranked[j].Rate {
			return ranked[i].Rate > ranked[j].Rate
		}
		if ranked[i].Count != ranked[j].Count {
			return ranked[i].Count > ranked[j].Count
		}
		if ranked[i].Diff.Char != ranked[j].Diff.Char {
			return ranked[i].Diff.Char < ranked[j].Diff.Char
		}
		return ranked[i].Diff.Replace < ranked[j].Diff.Replace
	})
	n := len(ranked)
	var cum int
	for i := 0; i < n; {
		// Group ties so equal rates get the same percentile
		j := i
		for j < n && ranked[j].Rate == ranked[i].Rate {
			j++
		}
		pct := 100.0 * float64(n-j) / float64(n)
		for k := i; k < j; k++ {
			cum += ranked[k].Count
			ranked[k].Percentile = pct
		}
		i = j
	}
	for _, r := range ranked {
		fmt.Printf("%[1]c→%[2]c  %[1]U→%[2]U  %6[3]d/%-7[4]d rate=%5.1[5]f%%  pct=%5.1[6]f\n",
			r.Diff.Char, r.Diff.Replace, r.Count, r.CharCount, r.Rate, r.Percentile)
	}
}

func CreateCharReplaceMap(replace []ReplaceCount, minRank float64) map[rune]map[rune]float64 {
	var result = make(map[rune]map[rune]float64)
	for _, r := range replace {
		if r.Rate > minRank {
			inner, ok := result[r.Diff.Char]
			if !ok {
				inner = make(map[rune]float64)
				result[r.Diff.Char] = inner
			}
			inner[r.Diff.Replace] = r.Rate
		}
	}
	return result
}

func PairsCompare(verses []generic.Verse2, replaceMap map[rune]map[rune]float64) [][]diff.Diff {
	var result [][]diff.Diff
	for _, vs := range verses {
		difSlice := LineCompare(vs, replaceMap)
		result = append(result, difSlice)
	}
	return result
}

func LineCompare(verse generic.Verse2, replaceMap map[rune]map[rune]float64) []diff.Diff {
	var result []diff.Diff
	refText := strings.ReplaceAll(verse.Text(), "-", " ")
	asrText := strings.ReplaceAll(verse.ASRText, "-", " ")
	wDiff := diff.WordLevenshtein(refText, asrText)
	for _, w := range wDiff {
		if w.Type == diff.OpReplace {
			w = WordCompare(w, replaceMap)
		}
		result = append(result, w)
	}
	return result
}

func WordCompare(word diff.Diff, replaceMap map[rune]map[rune]float64) diff.Diff {
	for _, ch := range diff.CharLevenshtein(word.Text, word.Replace) {
		if ch.Type == diff.OpEqual {
			continue
		}
		if ch.Type == diff.OpReplace && CharCompare(ch, replaceMap).Type == diff.OpEqual {
			continue
		}
		if (ch.Type == diff.OpInsert || ch.Type == diff.OpDelete) && IsDiffIgnorable(ch.Char) {
			continue
		}
		return word // at least one difference is not an allowed replacement
	}
	return diff.Diff{Type: diff.OpEqual, Text: word.Text}
}

func CharCompare(ch diff.CDiff, replaceMap map[rune]map[rune]float64) diff.CDiff {
	_, ok := replaceMap[ch.Char][ch.Replace]
	if ok {
		return diff.CDiff{Type: diff.OpEqual, Char: ch.Char}
	}
	return ch
}

func CountErrorsCER(pairs []generic.Pair) float64 {
	var total int
	var errors int
	for _, p := range pairs {
		cdiff := diff.CharLevenshtein(p.Base.Text, p.Comp.Text)
		cnt := diff.Count(cdiff)
		total += cnt.RefLen
		errors += cnt.Substitutions + cnt.Deletions + cnt.Insertions
	}
	return float64(errors) / float64(total)
}

func IsDiffIgnorable(c rune) bool {
	return c == 0x09CD // BENGALI SIGN VIRAMA, Claude really recommended dropping this from end
}

func IsDDiffDigits(dif diff.Diff) bool {
	if !IsWordDigits(dif.Text) {
		return false
	}
	return IsWordDigits(dif.Replace)
}

func IsWordDigits(str string) bool {
	for _, ch := range str {
		if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

/*
U+09CD ্ BENGALI SIGN VIRAMA
When I talked about the "word-final hasanta", I meant this character at the end of a word,
as in গোজেনর্ vs গোজেনর and তার্ vs তার in the ccp report. The reference text is inconsistent
about it there, and you can't hear the difference.

Two nearby characters from the same report that may be what you're thinking of:
U+09CE ৎ BENGALI LETTER KHANDA TA, as in বৈৎলেহম. It is a word-final form of ত + ্ (U+09A4 U+09CD),
and spellings sometimes switch between the two. That makes it a likely candidate for your
equivalence table.
U+09BC ় BENGALI SIGN NUKTA. In your report য় appears as two characters, য (U+09AF)
plus ় (U+09BC), each with its own score. The single-character form U+09DF decomposes to
those two under NFC, so after your NFC normalization it will always be the two-character sequence.
It's worth knowing about, but not something to remove.

*/
