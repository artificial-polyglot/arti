package compare_rpt

import (
	"database/sql"
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/generic"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Pair struct {
	Ref       generic.VerseRef      `json:"ref"`
	ScriptNum string                `json:"script_num"`
	AudioFile string                `json:"audio_file"`
	BeginTS   float64               `json:"begin_ts"`
	EndTS     float64               `json:"end_ts"`
	Base      PairText              `json:"base"`
	Comp      PairText              `json:"comp"`
	Diffs     []diffmatchpatch.Diff `json:"diffs"`
	HTML      string                `json:"html"`
}

type PairText struct {
	ScriptId int    `json:"script_id"`
	Text     string `json:"text"`
	Uroman   string `json:"uroman"`
}

func NewPair(vs generic.Verse2, refText string, asrText string) Pair {
	var p Pair
	p.Ref.BookId = vs.LineRef.BookId
	p.Ref.ChapterNum = vs.LineRef.ChapterNum
	//p.Ref.ChapterEnd = base.chapterEnd
	p.Ref.VerseStr = vs.LineRef.VerseStr
	//p.Ref.VerseEnd = base.verseEnd
	//p.ScriptNum = base.ScriptNum
	p.BeginTS = vs.BeginTS
	p.EndTS = vs.EndTS
	p.Base.ScriptId = int(vs.ScriptId)
	p.Comp.ScriptId = int(vs.ScriptId)
	p.Base.Text = refText
	p.Base.Uroman = "" //TBD
	p.Comp.Text = asrText
	p.Comp.Uroman = "" //TBD
	return p
}

func (p *Pair) Text(isLatin sql.NullBool) (string, string) {
	if isLatin.Bool {
		return p.Base.Text, p.Comp.Text
	} else {
		return p.Base.Uroman, p.Comp.Uroman
	}
}

func (p *Pair) Inserts() int {
	var inserts int
	for _, diff := range p.Diffs {
		if diff.Type == diffmatchpatch.DiffInsert {
			inserts += utf8.RuneCountInString(diff.Text)
		}
	}
	return inserts
}

func (p *Pair) Deletes() int {
	var deletes int
	for _, diff := range p.Diffs {
		if diff.Type == diffmatchpatch.DiffDelete {
			deletes += utf8.RuneCountInString(diff.Text)
		}
	}
	return deletes
}

func (p *Pair) LargestLength() int {
	var result int
	var length int
	for _, diff := range p.Diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			length += utf8.RuneCountInString(diff.Text)
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
