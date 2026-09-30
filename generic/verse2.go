package generic

import "strings"

type Verse2 struct {
	ScriptId int64
	LineRef  VerseRef
	ASRText  string  // Transcript
	BeginTS  float64 // Used in report to play audio
	EndTS    float64 // Used in report to play audio
	Duration float64 // Displayed in audio
	Words    []Word2
}
type Word2 struct {
	WordId    int64
	Chars     []Char2
	Text      string // Word text with no punctuation
	WordPunct string // Word with punctuation added back
	Uroman    string
	BeginTS   float64 // Used in report for word Karoke highlighting
	EndTS     float64 // Used in report for word Karoke highlighting
	FAScore   float64 // Computed in align_compare
	IsASR     bool
	Opacity   float64
}

// Char2 is sourced from chars_qa_align, words_qa_align, and scripts_qa_align,
// the result of forced alignment using a model for the language, not MMS.
// qa_align adds each verse's chop offset (qa_align.go's FARequest.BeginTS)
// onto its timestamps before storing them, so they are chapter-absolute, the
// same frame as the timestamps in the other tables.
type Char2 struct {
	Char        rune
	BeginTS     float64 // Used in align_silence to compute silence
	EndTS       float64 // Used in align_silence to compute silence
	FAScore     float64 // The critical output from qa_align used to locate errors
	Silence     float64 // Computed in align_silence
	SilenceLong int
	IsASR       bool
}

// ToPair builds a Pair whose Base is this verse's lowercased word text and
// whose Comp is compText, attributed to compScriptId.
func (v Verse2) ToPair() Pair {
	var p Pair
	p.Ref.BookId = v.LineRef.BookId
	p.Ref.ChapterNum = v.LineRef.ChapterNum
	p.Ref.VerseStr = v.LineRef.VerseStr
	p.BeginTS = v.BeginTS
	p.EndTS = v.EndTS
	p.Base.ScriptId = v.ScriptId
	p.Base.Text = v.Text()
	p.Base.Uroman = v.Uroman()
	p.Comp.ScriptId = v.ScriptId
	p.Comp.Text = v.ASRText
	return p
}

// Text returns the verse's words joined by spaces, lowercased.
func (v Verse2) Text() string {
	var text []string
	for _, wd := range v.Words {
		text = append(text, wd.Text)
	}
	return strings.ToLower(strings.Join(text, " "))
}

func (v Verse2) Uroman() string {
	var text []string
	for _, wd := range v.Words {
		text = append(text, wd.Uroman)
	}
	return strings.ToLower(strings.Join(text, " "))
}
