package asr_align

import "fmt"

type Char struct {
	ScriptId int64
	WordId   int64
	VerseStr string
	WordSeq  int
	CharSeq  int  // questionable need
	Char     rune `json:"ch"`
	BeginTS  float64
	EndTS    float64 `json:"ts"`
}

const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

/**
Green: Has Verse and Timestamp
Blue: Has Verse, but no Timestamp
Yellow: Has no Verse, but has Timestamp
Red: Has neither
*/

func (c Char) Colored() string {
	if c.VerseStr != "" && c.EndTS != 0.0 {
		return Green + string(c.Char) + Reset
	} else if c.VerseStr != "" && c.EndTS == 0.0 {
		return Blue + string(c.Char) + Reset
	} else if c.VerseStr == "" && c.EndTS != 0.0 {
		return Yellow + string(c.Char) + Reset
	} else {
		return Red + string(c.Char) + Reset
	}
}

func DumpChars(chars []Char) {
	for _, c := range chars {
		fmt.Print(c.Colored())
	}
	fmt.Println()
}
