package generic

import "strings"

type Verse2 struct {
	ScriptId    int64    `json:"script_id"`
	LineRef     VerseRef `json:"verse_ref"`
	ASRText     string   `json:"asr_transcript"` // Transcript
	BeginTS     float64  `json:"begin_ts"`       // Start of the audio given to ASR and qa_align
	EndTS       float64  `json:"end_ts"`         // End of the audio given to ASR and qa_align
	PlayBeginTS float64  `json:"play_begin_ts"`  // Used in report to play audio, see scripts.play_begin_ts
	PlayEndTS   float64  `json:"play_end_ts"`    // Used in report to play audio
	Duration    float64  `json:"duration"`       // Displayed in audio
	Words       []Word2  `json:"words"`
}
type Word2 struct {
	WordId    int64   `json:"word_id"`
	Chars     []Char2 `json:"chars"`
	Text      string  `json:"word"`       // Word text with no punctuation
	WordPunct string  `json:"word_punct"` // Word with punctuation added back
	Uroman    string  `json:"uroman"`
	BeginTS   float64 `json:"begin_ts"` // Used in report for word Karoke highlighting
	EndTS     float64 `json:"end_ts"`   // Used in report for word Karoke highlighting
	FAScore   float64 `json:"fa_score"` // Computed in align_compare
	IsASR     bool    `json:"-"`
	Opacity   float64 `json:"-"`
}

// Char2 is sourced from chars_qa_align, words_qa_align, and scripts_qa_align,
// the result of forced alignment using a model for the language, not MMS.
// qa_align adds each verse's chop offset (qa_align.go's FARequest.BeginTS)
// onto its timestamps before storing them, so they are chapter-absolute, the
// same frame as the timestamps in the other tables.
type Char2 struct {
	Char        rune    `json:"char"`
	BeginTS     float64 `json:"begin_ts"` // Used in align_silence to compute silence
	EndTS       float64 `json:"end_ts"`   // Used in align_silence to compute silence
	FAScore     float64 `json:"fa_score"` // The critical output from qa_align used to locate errors
	Silence     float64 `json:"-"`        // Computed in align_silence
	SilenceLong int     `json:"-"`
	IsASR       bool    `json:"-"`
}

// PlayTimes returns the audio a report plays for this verse.  A database aligned before the
// play times existed has none, and then plays the verse's ASR audio.
func (v Verse2) PlayTimes() (float64, float64) {
	if v.PlayEndTS > v.PlayBeginTS {
		return v.PlayBeginTS, v.PlayEndTS
	}
	return v.BeginTS, v.EndTS
}

// ToPair builds a Pair whose Base is this verse's lowercased word text and
// whose Comp is compText, attributed to compScriptId.
func (v Verse2) ToPair() Pair {
	var p Pair
	p.Ref.BookId = v.LineRef.BookId
	p.Ref.ChapterNum = v.LineRef.ChapterNum
	p.Ref.VerseStr = v.LineRef.VerseStr
	p.BeginTS, p.EndTS = v.PlayTimes()
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
