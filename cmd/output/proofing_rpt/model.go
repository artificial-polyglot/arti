package proofing_rpt

import "github.com/artificial-polyglot/arti/generic"

type Verse2 struct {
	ScriptId int64
	LineRef  generic.VerseRef
	BeginTS  float64 // Used in report to play audio
	EndTS    float64 // Used in report to play audio
	Duration float64 // Displayed in audio
	Words    []Word2
}
type Word2 struct {
	WordId    int64
	Ttype     string // W=word, ASR=added by align_compare
	Chars     []Char2
	Text      string // Word text with no punctuation
	WordPunct string // Word with punctuation added back
	Uroman    string
	BeginTS   float64 // Used in report for word Karoke highlighting
	EndTS     float64 // Used in report for word Karoke highlighting
	FAScore   float64 // Computed in align_compare
	Opacity   float64
}

// Char2 is sourced from chars_qa_align which is the result of forced alignment
// using a model for the language, not MMS.  The timestamps here are per verse,
// while the timestamps in the other tables are per chapter.
type Char2 struct {
	Char        rune
	BeginTS     float64 // Used in align_silence to compute silence
	EndTS       float64 // Used in align_silence to compute silence
	FAScore     float64 // The critical output from qa_align used to locate errors
	Silence     float64 // Computed in align_silence
	SilenceLong int
	IsASR       bool
}
