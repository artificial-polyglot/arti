package generic

import (
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
	for _, df := range p.Text.Diffs {
		if df.Type == diff.OpInsert {
			inserts += utf8.RuneCountInString(df.Text)
		}
	}
	return inserts
}

func (p *Pair) Deletes() int {
	var deletes int
	for _, df := range p.Text.Diffs {
		if df.Type == diff.OpDelete {
			deletes += utf8.RuneCountInString(df.Text)
		}
	}
	return deletes
}

func (p *Pair) LargestLength() int {
	var result int
	var length int
	for _, df := range p.Text.Diffs {
		if df.Type != diff.OpEqual {
			length += utf8.RuneCountInString(df.Text)
		} else {
			if length > result {
				result = length
			}
			length = 0
		}
	}
	if length > result {
		result = length
	}
	return result
}

func (p *Pair) ErrorPct(inserts int, deletes int) float64 {
	avgLen := float64(len(p.Base.Uroman)+len(p.Comp.Uroman)) / 2.0
	return float64((inserts+deletes)*100) / avgLen
}
