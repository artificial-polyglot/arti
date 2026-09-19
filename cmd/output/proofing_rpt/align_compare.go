package proofing_rpt

import (
	"database/sql"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
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
			refText := a.GetOriginalText(verse)
			cDiffs := diff.CharDiff(refText, asrText)
			newLine := a.MarkDeletedChars(verse, cDiffs)
			newLine = a.InsertASRSilenceChars(newLine, cDiffs)
			result = append(result, newLine)
		}
	}
	//ComputeFAWordError(result)
	ComputeMinWordError(result)
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

func (a *AlignSilence) InsertASRSilenceChars(verse Verse2, cDiffs []diff.CDiff) Verse2 {
	newWords := make([]Word2, 0, len(verse.Words)+10)
	// ASR text inserted before the very first reference char has no preceding
	// word to follow, so the per-char scan below (which only looks forward
	// from a matched reference char) can never reach it. Handle it once,
	// up front, interpolated against the gap before the first real char.
	leading := a.buildASRWordFromInserts(cDiffs, 0)
	if len(leading.Chars) > 0 {
		a.interpolateASRTimestamps(&leading, verse.BeginTS, firstCharBeginTS(verse)-verse.BeginTS)
		newWords = append(newWords, leading)
	}
	position := -1
	for _, wd := range verse.Words {
		var pendingASR []Word2
		for _, ch := range wd.Chars {
			position++
			//if ch.SilenceLong > 0 {
			if true {
				diffPos := a.FindPositionInDiff(cDiffs, position)
				if diffPos >= 0 {
					newWord := a.buildASRWordFromInserts(cDiffs, diffPos+1)
					if len(newWord.Chars) > 0 {
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

// buildASRWordFromInserts consumes the run of consecutive DiffInsert entries
// in cDiffs starting at idx, returning the resulting ASR word (zero value,
// with no Chars, if idx isn't the start of an insert run).
func (a *AlignSilence) buildASRWordFromInserts(cDiffs []diff.CDiff, idx int) Word2 {
	var newWord Word2
	var text []rune
	for i := idx; i < len(cDiffs) && cDiffs[i].Type == diff.OpInsert; i++ {
		newChar := Char2{
			Char:    cDiffs[i].Char,
			BeginTS: -1,
			EndTS:   -1,
			FAScore: 1.1,
			IsASR:   true,
		}
		text = append(text, newChar.Char)
		newWord.Chars = append(newWord.Chars, newChar)
	}
	if len(newWord.Chars) > 0 {
		newWord.IsASR = true
		newWord.Text = string(text)
		newWord.FAScore = 1.1
	}
	return newWord
}

// firstCharBeginTS returns the BeginTS of the verse's first character, used
// with Verse2.BeginTS to size the gap before it for a leading ASR insertion.
// Both are chapter-absolute. Note verse.BeginTS (scripts_qa_align.begin_ts)
// is itself defined as the first word's begin, i.e. the first char's begin,
// so this gap is 0 in practice and the leading case degrades to the -1
// "unknown timestamp" fallback in interpolateASRTimestamps.
func firstCharBeginTS(verse Verse2) float64 {
	for _, wd := range verse.Words {
		if len(wd.Chars) > 0 {
			return wd.Chars[0].BeginTS
		}
	}
	return 0
}

// MarkDeletedChars walks the verse's chars in the same
// order as InsertASRSilenceChars, and for every reference char that
// diff-match-patch marked as Deleted - present in refText but absent from
// asrText, meaning the ASR transcript never produced it - sets that char's
// FAScore to 0.0. It repeats its own DiffMatchPatch call rather than sharing
// cDiffs with InsertASRSilenceChars, so the two can be tried independently.
func (a *AlignSilence) MarkDeletedChars(verse Verse2, cDiffs []diff.CDiff) Verse2 {
	position := -1
	for wi := range verse.Words {
		chars := verse.Words[wi].Chars
		for ci := range chars {
			position++
			diffPos := a.FindPositionInDiff(cDiffs, position)
			if diffPos < len(cDiffs) && cDiffs[diffPos].Type == diff.OpDelete {
				chars[ci].FAScore = -0.1
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

func (a *AlignSilence) FindPositionInDiff(cDiffs []diff.CDiff, charPos int) int {
	refCount := -1
	for i, ch := range cDiffs {
		if ch.Type != diff.OpInsert {
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
			//verses[i].Words[j].FAScore = faWord.MinScore
		}
	}
}

func ComputeMinWordError(verses []Verse2) {
	for i, vs := range verses {
		for j, wd := range vs.Words {
			var minimum = 1.0
			for c := range wd.Chars {
				ch := wd.Chars[c]
				if len(wd.Chars) > 4 {
					if c > 0 && c < len(wd.Chars)-1 {
						if minimum > ch.FAScore {
							minimum = ch.FAScore
						}
					}
				} else {
					if minimum > ch.FAScore {
						minimum = ch.FAScore
					}
				}
			}
			verses[i].Words[j].FAScore = minimum
		}
	}
}
