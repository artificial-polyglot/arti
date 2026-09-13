package proofing_rpt

import (
	"database/sql"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/fa"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (a *AlignSilence) CompareLines2ASR(verses []Verse2) ([]Verse2, *log.Status) {
	var result []Verse2
	var status *log.Status
	for _, verse := range verses {
		if false {
			//if !a.HasSilence(verse) {
			result = append(result, verse)
		} else {
			var asrText string
			asrText, status = a.SelectTranscript(verse.ScriptId)
			if status != nil {
				return result, status
			}
			refText := a.GetOriginalText(verse) // This could be done by selecting line
			cDiffs := a.DiffMatchPatch(refText, asrText)
			newLine := a.MarkDeletedChars(verse, cDiffs)
			newLine = a.InsertASRSilenceChars(newLine, cDiffs)
			result = append(result, newLine)
		}
	}
	ComputeFAWordError(result)
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

func (a *AlignSilence) InsertASRSilenceChars(verse Verse2, cDiffs []CDiff) Verse2 {
	newWords := make([]Word2, 0, len(verse.Words)+10)
	position := -1
	for _, wd := range verse.Words {
		var pendingASR []Word2
		for _, ch := range wd.Chars {
			position++
			//if ch.SilenceLong > 0 {
			if true {
				diffPos := a.FindPositionInDiff(cDiffs, position)
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
						newWord.IsASR = true
						newWord.Text = string(text)
						newWord.FAScore = 1.0
						a.interpolateASRTimestamps(&newWord, ch.EndTS, ch.Silence)
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

// MarkDeletedChars is an experiment: it walks the verse's chars in the same
// order as InsertASRSilenceChars, and for every reference char that
// diff-match-patch marked as Deleted - present in refText but absent from
// asrText, meaning the ASR transcript never produced it - sets that char's
// FAScore to 0.0. It repeats its own DiffMatchPatch call rather than sharing
// cDiffs with InsertASRSilenceChars, so the two can be tried independently.
func (a *AlignSilence) MarkDeletedChars(verse Verse2, cDiffs []CDiff) Verse2 {
	position := -1
	for wi := range verse.Words {
		chars := verse.Words[wi].Chars
		for ci := range chars {
			position++
			diffPos := a.FindPositionInDiff(cDiffs, position)
			if diffPos < len(cDiffs) && cDiffs[diffPos].Type == diffmatchpatch.DiffDelete {
				chars[ci].FAScore = 0.0
			}
		}
	}
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
			if ch != ' ' || df.Type == diffmatchpatch.DiffInsert {
				result = append(result, CDiff{Type: df.Type, Char: ch})
			}
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

func ComputeFAWordError(verses []Verse2) {
	for i, vs := range verses {
		for j, wd := range vs.Words {
			var faChars []fa.FAChar
			for _, ch := range wd.Chars {
				var char fa.FAChar
				char.Char = ch.Char
				char.BeginTS = ch.BeginTS
				char.EndTS = ch.EndTS
				char.FAScore = ch.FAScore
				char.Silence = ch.Silence
				char.SilenceLong = ch.SilenceLong
				char.IsASR = ch.IsASR
				faChars = append(faChars, char)
			}
			faWord := fa.ComputeWordFA(faChars, fa.DefaultFAConfig())
			verses[i].Words[j].FAScore = faWord.TrimmedMinScore
		}
	}
}
