package proofing_rpt

import (
	"database/sql"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (a *AlignSilence) CompareLines2ASR(verses []Verse2) ([]Verse2, *log.Status) {
	var result []Verse2
	var status *log.Status
	for _, verse := range verses {
		verse = a.InsertSpaces(verse)
		if !a.HasSilence(verse) {
			result = append(result, verse)
		} else {
			var asrText string
			asrText, status = a.SelectTranscript(verse.ScriptId)
			if status != nil {
				return result, status
			}
			alignedText := a.GetOriginalText(verse) // This could be done by selecting line
			newLine := a.insertASRSilenceChars(verse, alignedText, asrText)
			result = append(result, newLine)
		}
	}
	return result, status
}

func (a *AlignSilence) InsertSpaces(verse Verse2) Verse2 {
	var result Verse2
	result.ScriptId = verse.ScriptId
	result.LineRef = verse.LineRef
	result.BeginTS = verse.BeginTS
	result.EndTS = verse.EndTS
	result.Duration = verse.Duration

	lastWord := len(verse.Words) - 1
	for i := 0; i < len(verse.Words); i++ {
		word := verse.Words[i]
		if i < lastWord {
			var newChar Char2
			newChar.Char = ' '
			newChar.FAScore = 1.0
			word.Chars = append(word.Chars, newChar)
		}
		result.Words = append(result.Words, word)
	}
	return result
}

func (a *AlignSilence) HasSilence(verse Verse2) bool {
	for _, wd := range verse.Words {
		for _, char := range wd.Chars {
			if char.SilenceLong > 0 {
				return true
			}
		}
	}
	return false
}

func (a *AlignSilence) GetOriginalText(verse Verse2) string {
	var allChars []rune
	for _, wd := range verse.Words {
		for _, ch := range wd.Chars {
			allChars = append(allChars, ch.Char)
		}
	}
	return string(allChars)
}

func (a *AlignSilence) insertASRSilenceChars(verse Verse2, alignedText, asrText string) Verse2 {
	cDiffs := a.DiffMatchPatch(alignedText, asrText)
	var position int
	for i, wd := range verse.Words {
		for _, ch := range wd.Chars {
			if ch.SilenceLong > 0 {
				diffPos := a.FindPositionInDiff(cDiffs, position)
				for i := diffPos + 1; i < len(cDiffs) && cDiffs[i].Type == diffmatchpatch.DiffInsert; i++ {
					var newChar Char2
					newChar.Char = cDiffs[i].Char
					newChar.BeginTS = -1
					newChar.EndTS = -1
					newChar.FAScore = 1.0
					newChar.IsASR = true
					wd.Chars = append(wd.Chars, newChar)
				}
			}
			position += 1
		}
		verse.Words[i] = wd
	}
	return verse
}

type CDiff struct {
	Type diffmatchpatch.Operation
	Char rune
}

func (a *AlignSilence) DiffMatchPatch(text string, asrText string) []CDiff {
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

func (a *AlignSilence) SelectTranscript(scriptId int64) (string, *log.Status) {
	query := `SELECT transcript FROM scripts_qa_align WHERE script_id = ?`
	row := a.conn.DB.QueryRow(query, scriptId)
	var transcript string
	err := row.Scan(&transcript)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", log.Error(a.ctx, 500, err, "Failed to select from qa_align_scripts")
	} else {
		return transcript, nil
	}
}
