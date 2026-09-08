package generic

type AlignLine struct {
	Chars []AlignChar
}

type AlignChar struct {
	LineId      int64
	LineRef     VerseRef // e.g. GEN 1:3-5a or GEN 1 or GEN 1:49-2:1
	WordId      int64    // WordSeq, Word could be added
	Word        string
	WordPunct   string
	CharSeq     int
	Char        rune
	BeginTS     float64
	EndTS       float64
	FAScore     float64
	IsASR       bool
	Duration    float64
	Silence     float64
	SilencePos  int
	ScoreError  int
	SilenceLong int
}
