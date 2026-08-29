package proofing_rpt

import (
	"github.com/artificial-polyglot/arti/generic"
)

type Verse2 struct {
	ScriptId int64
	LineRef  generic.VerseRef
	BeginTS  float64
	EndTS    float64
	Duration float64
	Words    []Word2
}
type Word2 struct {
	WordId  int64
	Ttype   string
	Chars   []Char2
	Text    string
	Uroman  string
	BeginTS float64
	EndTS   float64
	FAScore float64
	Opacity float64
}

type Char2 struct {
	Char        rune
	BeginTS     float64
	EndTS       float64
	FAScore     float64
	Silence     float64
	SilenceLong int
	IsASR       bool
}

// PrepareDataForWriter groups a flat run of chars - ordered exactly by
// script_id, word_id, word_seq (see db.SelectFACharTimestamps) - into
// verses, then into words within each verse. Verse boundaries are detected
// from LineId changes and word boundaries from WordId changes in a single pass
func (a *AlignSilence) PrepareDataForWriter(chars []generic.AlignChar) []Verse2 {
	var verses []Verse2
	var verse Verse2
	var words []Word2
	var word Word2
	var sumFAScore, cntFAScore float64
	haveVerse := false
	haveWord := false

	flushWord := func() {
		if !haveWord {
			return
		}
		if cntFAScore > 0 {
			word.FAScore = sumFAScore / cntFAScore
		}
		words = append(words, word)
	}
	flushVerse := func() {
		if !haveVerse {
			return
		}
		flushWord()
		verse.Words = words
		if len(words) > 0 {
			verse.BeginTS = words[0].BeginTS
			verse.EndTS = words[len(words)-1].EndTS
			verse.Duration = verse.EndTS - verse.BeginTS
		}
		verses = append(verses, verse)
	}

	for _, char := range chars {
		if !haveVerse || char.LineId != verse.ScriptId {
			flushVerse()
			verse = Verse2{ScriptId: char.LineId, LineRef: char.LineRef}
			words = nil
			haveVerse = true
			haveWord = false // force a fresh word even if WordId repeats across the verse boundary
		}
		if !haveWord || char.WordId != word.WordId {
			flushWord()
			word = Word2{WordId: char.WordId, Ttype: "W", Text: char.Word, BeginTS: char.BeginTS}
			sumFAScore, cntFAScore = 0, 0
			haveWord = true
		}
		char2 := Char2{Char: char.Char, BeginTS: char.BeginTS, EndTS: char.EndTS,
			FAScore: char.FAScore, Silence: char.Silence, SilenceLong: char.SilenceLong}
		word.Chars = append(word.Chars, char2)
		word.EndTS = char.EndTS
		sumFAScore += char.FAScore
		cntFAScore++
	}
	flushVerse()

	return verses
}
