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
		if !a.HasSilence(verse) {
			result = append(result, verse)
		} else {
			var asrText string
			asrText, status = a.SelectTranscript(verse.ScriptId)
			if status != nil {
				return result, status
			}
			refText := a.GetOriginalText(verse) // This could be done by selecting line
			newLine := a.InsertASRSilenceChars(verse, refText, asrText)
			result = append(result, newLine)
		}
	}
	return result, status
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
	var text []string
	for _, wd := range verse.Words {
		text = append(text, wd.Text)
	}
	return strings.ToLower(strings.Join(text, " "))
}

func (a *AlignSilence) InsertASRSilenceChars(verse Verse2, refText, asrText string) Verse2 {
	cDiffs := a.DiffMatchPatch(refText, asrText)
	newWords := make([]Word2, 0, len(verse.Words)+10)
	position := -2
	for _, wd := range verse.Words {
		position++
		var pendingASR []Word2
		for _, ch := range wd.Chars {
			position++
			//if ch.SilenceLong > 0 {
			if true {
				lookupPos := position
				if ch.SilenceLong == int(betweenWordsLong) {
					// GetOriginalText inserts a synthetic space between this word
					// and the next; its own diff entry must be passed before
					// scanning for ASR inserts, or an insertion right after the
					// space is missed entirely.
					lookupPos++
				}
				diffPos := a.FindPositionInDiff(cDiffs, lookupPos)
				if diffPos >= 0 {
					var newWord Word2
					var text []rune
					for i := diffPos + 1; i < len(cDiffs) && cDiffs[i].Type == diffmatchpatch.DiffInsert; i++ {
						newChar := Char2{
							Char:    cDiffs[i].Char,
							BeginTS: -1,
							EndTS:   -1,
							FAScore: 1.0,
							IsASR:   true,
						}
						text = append(text, newChar.Char)
						newWord.Chars = append(newWord.Chars, newChar)
					}
					if len(newWord.Chars) > 0 {
						a.interpolateASRTimestamps(&newWord, ch.EndTS, ch.Silence)
						newWord.IsASR = true
						newWord.Text = string(text)
						newWord.FAScore = 1.0
						pendingASR = append(pendingASR, newWord)
					}
				}
			}
		}
		newWords = append(newWords, wd)
		newWords = append(newWords, pendingASR...) // ASR words follow their word
	}
	verse.Words = newWords
	return verse
}

// interpolateASRTimestamps assigns BeginTS/EndTS to each inserted ASR char by
// dividing the silence gap (span) that follows the reference char at start
// evenly across them, and sets newWord's own BeginTS/EndTS from the result.
// When there's no usable gap (span <= 0), every timestamp falls back to the
// -1 "unknown" sentinel instead.
func (a *AlignSilence) interpolateASRTimestamps(newWord *Word2, start, span float64) {
	n := len(newWord.Chars)
	if span > 0 {
		slice := span / float64(n)
		for j := range newWord.Chars {
			newWord.Chars[j].BeginTS = start + float64(j)*slice
			newWord.Chars[j].EndTS = start + float64(j+1)*slice
		}
		newWord.BeginTS = newWord.Chars[0].BeginTS
		newWord.EndTS = newWord.Chars[n-1].EndTS
	} else {
		for j := range newWord.Chars {
			newWord.Chars[j].BeginTS = -1
			newWord.Chars[j].EndTS = -1
		}
		newWord.BeginTS = -1
		newWord.EndTS = -1
	}
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
	refCount := -1
	for i, ch := range cDiffs {
		if ch.Type != diffmatchpatch.DiffInsert {
			refCount++
		}
		if refCount >= charPos {
			return i // array index of the charPos-th reference char
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
