package align

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
			newChar.Uroman = ' '
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
	var alUroman []rune
	for _, char := range chars {
		alUroman = append(alUroman, char.Uroman)
	}
	return string(alUroman)
}

func (a *AlignSilence) insertASRSilenceChars(
	lineRef string,
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
				Uroman:    cDiffs[i].Char,
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

func (a *AlignSilence) DiffMatchPatch(lineRef string, text string, asrText string) []CDiff {
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
