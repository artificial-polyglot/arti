package proofing_rpt

import (
	"strings"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (a *AlignSilence) compareLines2ASR(lines []generic.AlignLine, asrConn db.DBAdapter) ([]generic.AlignLine, *log.Status) {
	var result []generic.AlignLine
	var status *log.Status
	for _, line := range lines {
		line.Chars = a.InsertSpaces(line.Chars)
		var silencePos = a.FindSilencePos(line.Chars)
		if len(silencePos) == 0 {
			result = append(result, line)
		} else {
			//result = append(result, line) // Duplicate line for debugging
			lineId := line.Chars[0].LineId
			lineRef := line.Chars[0].LineRef
			var asrText string
			asrText, status = asrConn.SelectUromanLine(lineId)
			if status != nil {
				return result, status
			}
			alignedText := a.GetOriginalText(line.Chars) // This could be done by selecting line
			//fmt.Println(len(alignUroman))
			newLine := a.insertASRSilenceChars(lineRef, line, alignedText, asrText, silencePos)
			result = append(result, newLine)
		}
	}
	return result, status
}

func (a *AlignSilence) InsertSpaces(chars []generic.AlignChar) []generic.AlignChar {
	var result []generic.AlignChar
	for i, char := range chars {
		if i > 0 && char.CharSeq == 0 {
			var newChar generic.AlignChar
			newChar.AudioFile = char.AudioFile
			newChar.LineId = char.LineId
			newChar.LineRef = char.LineRef
			newChar.Char = ' '
			newChar.FAScore = 1.0
			result = append(result, newChar)
		}
		result = append(result, char)

	}
	return result
}

func (a *AlignSilence) FindSilencePos(chars []generic.AlignChar) []int {
	var silencePos []int
	for i, char := range chars {
		if char.SilenceLong > 0 {
			silencePos = append(silencePos, i)
		}
	}
	return silencePos
}

func (a *AlignSilence) GetOriginalText(chars []generic.AlignChar) string {
	var allChars []rune
	for _, char := range chars {
		allChars = append(allChars, char.Char)
	}
	return string(allChars)
}

func (a *AlignSilence) insertASRSilenceChars(
	lineRef generic.VerseRef,
	line generic.AlignLine,
	alignedText, asrText string,
	silencePos []int,
) generic.AlignLine {

	cDiffs := a.DiffMatchPatch(lineRef, alignedText, asrText)

	var newLine generic.AlignLine
	silStart := 0

	for _, silPos := range silencePos {
		// 1. carry over original chars through the silence anchor
		for i := silStart; i <= silPos; i++ {
			newLine.Chars = append(newLine.Chars, line.Chars[i])
		}
		silStart = silPos + 1

		// 2. splice in the ASR inserts that fill the gap after curr
		curr := line.Chars[silPos]
		diffPos := a.FindPositionInDiff(cDiffs, silPos)
		for i := diffPos + 1; i < len(cDiffs) && cDiffs[i].Type == diffmatchpatch.DiffInsert; i++ {
			newLine.Chars = append(newLine.Chars, generic.AlignChar{
				AudioFile: curr.AudioFile,
				LineId:    curr.LineId,
				LineRef:   curr.LineRef,
				Char:      cDiffs[i].Char,
				BeginTS:   curr.EndTS,
				EndTS:     curr.EndTS + curr.Silence,
				FAScore:   1.0,
				IsASR:     true,
			})
		}
	}

	// 3. copy the tail: everything after the last silence
	for i := silStart; i < len(line.Chars); i++ {
		newLine.Chars = append(newLine.Chars, line.Chars[i])
	}

	return newLine
}

type CDiff struct {
	Type diffmatchpatch.Operation
	Char rune
}

func (a *AlignSilence) DiffMatchPatch(lineRef generic.VerseRef, text string, asrText string) []CDiff {
	var result []CDiff
	diffMatch := diffmatchpatch.New()
	text = strings.TrimSpace(text)
	asrText = strings.TrimSpace(asrText)
	diffs := diffMatch.DiffMain(text, asrText, false)
	diffs = diffMatch.DiffCleanupSemantic(diffs)
	//fmt.Println(lineRef, asrText)
	//fmt.Println(lineRef, text)
	//fmt.Println(lineRef, diffMatch.DiffPrettyText(diffs))
	//fmt.Println(diffs)
	for _, df := range diffs {
		for _, ch := range df.Text {
			var cDiff CDiff
			cDiff.Type = df.Type
			cDiff.Char = ch
			result = append(result, cDiff)
		}
	}
	return result
}

func (a *AlignSilence) FindPositionInDiff(cDiffs []CDiff, charPos int) int {
	var diffPos = -1
	for i, ch := range cDiffs {
		if ch.Type != diffmatchpatch.DiffInsert {
			diffPos++
			if diffPos >= charPos {
				return i
			}
		}
	}
	return len(cDiffs)
}

/*
type Verse struct {
	ScriptId int64
	LineRef  generic.VerseRef //LineRef in alignline is string
	BeginTS  float64
	EndTS    float64
	Duration float64
	Words    []Word
}
type Word struct {
	WordId  int64
	Text    string
	Uroman  string
	BeginTS float64
	EndTS   float64
	FAScore float64
}

word_id INTEGER PRIMARY KEY AUTOINCREMENT,
script_id INTEGER NOT NULL,
word_seq INTEGER NOT NULL,
verse_num INTEGER NOT NULL,
ttype TEXT NOT NULL DEFAULT 'W',
word TEXT NOT NULL,
uroman TEXT NOT NULL DEFAULT '',
word_begin_ts REAL NOT NULL DEFAULT 0.0,
word_end_ts REAL NOT NULL DEFAULT 0.0,
fa_score REAL NOT NULL DEFAULT 0.0,

type Word struct {
	VerseStr    string
	WordId      int
	ScriptId    int
	WordSeq     int
	VerseNum    int
	TType       string
	Word        string
	WordBeginTS float64
	WordEndTS   float64
	FAScore     float64
	WordEncoded []float64

func (a *AlignSilence) PrepareDataForWriter(lines []generic.AlignLine) []Verse {
	var verses = make([]Verse, 0, len(lines))
	var lastWordId = int64(-1)
	var sumFAScore, cntFAScore float64
	var word Word
	for _, line := range lines {
		var verse Verse
		verse.ScriptId = line.Chars[0].LineId
		verse.LineRef = generic.NewVerseRef(line.Chars[0].LineRef)
		verse.BeginTS = line.Chars[0].BeginTS
		for _, char := range line.Chars {
			var words = make([]Word, 0, 32)
			if char.WordId != lastWordId {
				words = append(words, word)
				verse.Words = words
				word = Word{}
				word.WordId = char.WordId
				word.Text = append(word.Text, char.Uroman)
				word.Uroman = ''
				word.BeginTS = char.BeginTS
				word.FAScore = sumFAScore / cntFAScore
				sumFAScore = 0.0
				cntFAScore = 0.0
			}
			verse.EndTS = char.EndTS
			word.EndTS = char.EndTS
			sumFAScore += char.FAScore
			cntFAScore += 1

		}
	}
}
	With QAAlign, the text data coming out of the ASR process is NOT uroman, but it is original text.
		This is a major difference with the align_compare code.  Uroman should only be provided on a while verse basis
	The inserted character data is not in the
*/
