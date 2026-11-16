package generic

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/utility/diff"
)

type Pair struct {
	Ref       VerseRef `json:"ref"`
	ScriptNum string   `json:"script_num"`
	AudioFile string   `json:"audio_file"`
	BeginTS   float64  `json:"begin_ts"`
	EndTS     float64  `json:"end_ts"`
	Base      PairText `json:"base"`
	Comp      PairText `json:"comp"`
	Text      PairDiff `json:"text"`
	Uroman    PairDiff `json:"uroman"`
}

type PairText struct {
	ScriptId int64  `json:"script_id"`
	Text     string `json:"text"`
	Uroman   string `json:"uroman"`
}

type PairDiff struct {
	Diffs []diff.Diff `json:"diffs"`
	HTML  string      `json:"html"`
}

func (p *Pair) Inserts() int {
	var inserts int
	if p.Ref.VerseStr != "0" {
		for _, df := range p.Text.Diffs {
			if df.Type == diff.OpInsert {
				inserts += utf8.RuneCountInString(df.Text)
			}
		}
	}
	return inserts
}

func (p *Pair) Deletes() int {
	var deletes int
	if p.Ref.VerseStr != "0" {
		for _, df := range p.Text.Diffs {
			if df.Type == diff.OpDelete {
				deletes += utf8.RuneCountInString(df.Text)
			}
		}
	}
	return deletes
}

// LargestLength returns the longest run of contiguous non-equal chars: inserts,
// deletes and replaces next to each other, ended by any equal text. Spaces are
// not counted. A replace contributes the longer of its two sides.
func (p *Pair) LargestLength() int {
	var result, length int
	for _, df := range p.Text.Diffs {
		if df.Type == diff.OpEqual {
			length = 0
			continue
		}
		n := countNonSpace(df.Text)
		if df.Type == diff.OpReplace {
			n = max(n, countNonSpace(df.Replace))
		}
		length += n
		result = max(result, length)
	}
	return result
}

// countNonSpace returns the number of chars in s that are not white space.
func countNonSpace(s string) int {
	var n int
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// LargestLengthTest returns the longest run of inserted/deleted chars. A single
// equal char between changes does not break the run, and is not counted.
func (p *Pair) LargestLengthTest() int {
	var result int
	var run strings.Builder
	endRun := func() {
		length := utf8.RuneCountInString(strings.TrimSpace(run.String()))
		if length > result {
			result = length
		}
		run.Reset()
	}
	for _, df := range p.Text.Diffs {
		if df.Type != diff.OpEqual {
			run.WriteString(df.Text)
		} else if utf8.RuneCountInString(df.Text) > 1 {
			endRun()
		}
	}
	endRun()
	return result
}

func (p *Pair) ErrorPct(inserts int, deletes int) float64 {
	avgLen := float64(len(p.Base.Uroman)+len(p.Comp.Uroman)) / 2.0
	return float64((inserts+deletes)*100) / avgLen
}
