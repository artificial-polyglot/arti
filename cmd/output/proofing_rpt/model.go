package proofing_rpt

import "github.com/artificial-polyglot/arti/generic"

type Verse2 struct {
	ScriptId int64
	LineRef  generic.VerseRef
	BeginTS  float64
	EndTS    float64
	Duration float64
	Words    []Word2
}
type Word2 struct {
	WordId    int64
	Ttype     string
	Chars     []Char2
	Text      string
	WordPunct string
	Uroman    string
	BeginTS   float64
	EndTS     float64
	FAScore   float64
	Opacity   float64
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
