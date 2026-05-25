package align

import (
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/sergi/go-diff/diffmatchpatch"
	"strings"
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
			var newLine generic.AlignLine
			lineId := line.Chars[0].LineId
			lineRef := line.Chars[0].LineRef
			var asrText string
			asrText, status = asrConn.SelectUromanLine(lineId)
			if status != nil {
				return result, status
			}
			alignedText := a.GetOriginalText(line.Chars)
			//fmt.Println(len(alignUroman))
			cDiffs := a.DiffMatchPatch(lineRef, alignedText, asrText)
			var silStart = 0
			for _, silPos := range silencePos {
				for i := silStart; i <= silPos; i++ {
					newLine.Chars = append(newLine.Chars, line.Chars[i])
					silStart = i + 1
				}
				curr := line.Chars[silPos]
				diffPos := a.FindPositionInDiff(cDiffs, silPos)
				//fmt.Println(silPos, string(alignNorm[silPos]), string(cDiffs[diffPos].Char))
				for i := diffPos + 1; i < len(cDiffs); i++ {
					if cDiffs[i].Type == diffmatchpatch.DiffInsert {
						//fmt.Println("add char ASR char", string(cDiffs[i].Char))
						var newChar generic.AlignChar
						newChar.AudioFile = curr.AudioFile
						newChar.LineId = curr.LineId
						newChar.LineRef = curr.LineRef
						newChar.Uroman = cDiffs[i].Char
						newChar.BeginTS = curr.EndTS
						newChar.EndTS = curr.EndTS + curr.Silence
						newChar.FAScore = 1.0
						newChar.IsASR = true
						newLine.Chars = append(newLine.Chars, newChar)
					} else {
						break
					}
				}
				//result = append(result, newLine)
			}
			for i := silencePos[len(silencePos)-1] + 1; i < len(line.Chars); i++ {
				newLine.Chars = append(newLine.Chars, line.Chars[i])
			}
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
