package diff

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/generic"
)

const minCharCount = 3
const minPctRank = 5.0

type CharFilter1 struct {
	replaceCounts  []ReplaceCount
	charReplaceMap map[rune]map[rune]float64
}

func NewCharFilter1(verses []generic.Verse2) CharFilter1 {
	var f CharFilter1
	charCount := CountCharOccurances(verses)
	countReplace := FindSimilarChars(verses)
	f.replaceCounts = CreateReplaceCounts(charCount, countReplace, minCharCount)
	//RankReplaceCounts(ranked)
	f.charReplaceMap = CreateCharReplaceMap(f.replaceCounts, minPctRank)
	return f
}

func (f *CharFilter1) DiffFilter(diffs []Diff) []Diff {
	for i := range diffs {
		df := diffs[i]
		if df.Type == OpReplace {
			diffs[i] = f.WordCompare(df)
		} else if df.Type == OpInsert || df.Type == OpDelete {
			if utf8.RuneCountInString(df.Text) == 1 {
				run, _ := utf8.DecodeRuneInString(df.Text)
				if IsDiffIgnorable(run) {
					diffs[i].Type = OpEqual
				}
			}
		}
	}
	return diffs
}

func (f *CharFilter1) VerseSliceCompare(verses []generic.Verse2) [][]Diff {
	var result [][]Diff
	for _, vs := range verses {
		difSlice := f.VerseCompare(vs)
		result = append(result, difSlice)
	}
	return result
}

func (f *CharFilter1) VerseCompare(verse generic.Verse2) []Diff {
	var result []Diff
	refText := strings.ReplaceAll(verse.Text(), "-", " ")
	asrText := strings.ReplaceAll(verse.ASRText, "-", " ")
	wDiff := WordLevenshtein(refText, asrText)
	for _, w := range wDiff {
		if w.Type == OpReplace {
			w = f.WordCompare(w)
		}
		result = append(result, w)
	}
	return result
}

func (f *CharFilter1) WordCompare(word Diff) Diff {
	for _, ch := range CharLevenshtein(word.Text, word.Replace) {
		if ch.Type == OpEqual {
			continue
		}
		if ch.Type == OpReplace && f.CharCompare(ch).Type == OpEqual {
			continue
		}
		if (ch.Type == OpInsert || ch.Type == OpDelete) && IsDiffIgnorable(ch.Char) {
			continue
		}
		return word // at least one difference is not an allowed replacement
	}
	return Diff{Type: OpEqual, Text: word.Text}
}

func (f *CharFilter1) CharCompare(ch CDiff) CDiff {
	_, ok := f.charReplaceMap[ch.Char][ch.Replace]
	if ok {
		return CDiff{Type: OpEqual, Char: ch.Char}
	}
	return ch
}

func (f *CharFilter1) DisplayCharFilter() {
	for c, m := range f.charReplaceMap {
		for r, val := range m {
			fmt.Println("From:", string(c), "To:", string(r), val)
		}
	}
}

func (f *CharFilter1) RankReplaceCounts() {
	// Highest rate first; then higher count, then char, so map order doesn't change the output
	sort.Slice(f.replaceCounts, func(i, j int) bool {
		if f.replaceCounts[i].Rate != f.replaceCounts[j].Rate {
			return f.replaceCounts[i].Rate > f.replaceCounts[j].Rate
		}
		if f.replaceCounts[i].Count != f.replaceCounts[j].Count {
			return f.replaceCounts[i].Count > f.replaceCounts[j].Count
		}
		if f.replaceCounts[i].Diff.Char != f.replaceCounts[j].Diff.Char {
			return f.replaceCounts[i].Diff.Char < f.replaceCounts[j].Diff.Char
		}
		return f.replaceCounts[i].Diff.Replace < f.replaceCounts[j].Diff.Replace
	})
	n := len(f.replaceCounts)
	var cum int
	for i := 0; i < n; {
		// Group ties so equal rates get the same percentile
		j := i
		for j < n && f.replaceCounts[j].Rate == f.replaceCounts[i].Rate {
			j++
		}
		pct := 100.0 * float64(n-j) / float64(n)
		for k := i; k < j; k++ {
			cum += f.replaceCounts[k].Count
			f.replaceCounts[k].Percentile = pct
		}
		i = j
	}
	for _, r := range f.replaceCounts {
		fmt.Printf("%[1]c→%[2]c  %[1]U→%[2]U  %6[3]d/%-7[4]d rate=%5.1[5]f%%  pct=%5.1[6]f\n",
			r.Diff.Char, r.Diff.Replace, r.Count, r.CharCount, r.Rate, r.Percentile)
	}
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

func FindSimilarChars(verses []generic.Verse2) map[CDiff]int {
	var countReplace = make(map[CDiff]int)
	for _, vs := range verses {
		cdiff := CharLevenshtein(vs.Text(), vs.ASRText)
		for _, d := range cdiff {
			if d.Type == OpReplace {
				countReplace[d]++
			}
		}
	}
	return countReplace
}

type ReplaceCount struct {
	Diff       CDiff
	Count      int     // times Char was replaced by Replace
	CharCount  int     // times Char occurs in the reference text
	Rate       float64 // % of Char occurrences replaced by Replace
	Percentile float64 // % of distinct pairs with a lower rate
	//CumShare   float64 // % of all replacements covered by this pair and all above it
}

func CreateReplaceCounts(charCounts map[rune]int, diffCounts map[CDiff]int, minCount int) []ReplaceCount {
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

func IsDiffIgnorable(c rune) bool {
	return c == 0x09CD // BENGALI SIGN VIRAMA, Claude really recommended dropping this from end
}
