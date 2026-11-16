package filters

import (
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/artificial-polyglot/arti/utility/fa"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// A check of the words of the text that are missing from the ASR transcript.  A word can be missing
// because the reader left it out, or because the trained model could not hear it.  The trained
// model's forced alignment (qa_align) cannot tell these apart, because the same model made the
// transcript.  mms_align's forced alignment uses a different model, the base MMS model, on the same
// audio: when a word really is missing from the audio, it has nowhere to put the word, and its
// fa_score is near 0.  When the reader said the word, it usually scores it normally.
//
// Only whole words are checked: every letter of the word must be deleted in the character diff.
// When some letters are deleted, it is usually an ASR spelling error, not missing audio.  Words with
// digits are skipped, because neither model can align digits that the reader says as words.
//
// A confirmed word stays missing, and is reported as a reading error.  A doubtful word, one the base
// model found in the audio, is taken as an ASR error and restored to the transcript where the diff
// deleted it, so later reports do not show it as a difference.  Run it after FilterAlignmentErrors, so
// that words moved to another verse are not reported as missing, and after the hallucination filter,
// so that a restored word cannot be removed again as an insert.

// MMSLimit is the most mms_align fa_score of a missing word that is confirmed.  In N1SKNSEC, missing
// words scored either 0.175 or less, or 0.61 or more, and 11.9% of words that ASR heard score below it.
const MMSLimit = 0.2

// MissingWord is one word of the text that is wholly missing from the ASR transcript.
type MissingWord struct {
	ScriptId  int64
	LineRef   generic.VerseRef
	WordIndex int
	Text      string
	QAScore   float64 // geometric mean of the qa_align scores of the word's chars, from the trained model
	MMSScore  float64 // mms_align fa_score of the word, from the base model
	Confirmed bool    // mms_align finds the word missing from the audio; otherwise it is restored
}

func (m MissingWord) Reason() string {
	score := strconv.FormatFloat(m.MMSScore, 'f', 2, 64)
	if m.Confirmed {
		return "missing from audio: mms_align score " + score
	}
	return "restored: mms_align score " + score + " says the word was read"
}

// FilterMissingWords returns every whole word missing from a transcript, with the decision made about
// it, and the verses with each doubtful word restored to the transcript.  The input verses are not
// changed.  verses must have qa_align chars and ASR transcripts, as proofing_rpt.SelectCharData and
// AddASRTranscript return them.
func FilterMissingWords(conn db.DBAdapter, verses []generic.Verse2) ([]generic.Verse2, []MissingWord, *log.Status) {
	scores, status := SelectMMSScores(conn)
	if status != nil {
		return verses, nil, status
	}
	missing := FindMissingWords(verses, scores)
	return RestoreWords(verses, missing), missing, nil
}

// RestoreWords returns a copy of verses with every missing word that is not confirmed put back into
// its transcript.
func RestoreWords(verses []generic.Verse2, missing []MissingWord) []generic.Verse2 {
	restore := make(map[int64]map[int]bool) // word indexes to restore, by script id
	for _, m := range missing {
		if m.Confirmed {
			continue
		}
		if restore[m.ScriptId] == nil {
			restore[m.ScriptId] = make(map[int]bool)
		}
		restore[m.ScriptId][m.WordIndex] = true
	}
	result := slices.Clone(verses)
	for i, vs := range result {
		if words, ok := restore[vs.ScriptId]; ok {
			result[i].ASRText = restoreVerse(vs, words)
		}
	}
	return result
}

// restoreVerse rebuilds the verse's transcript from the character diff, keeping what ASR wrote and
// adding back the deleted letters of the words in restore.  The spaces deleted by the diff are not
// used; a restored word is separated by a space from whatever is next to it.  The letters are counted
// the same way as in deletedLetters, so the k-th letter of the text belongs to word owner[k].
func restoreVerse(vs generic.Verse2, restore map[int]bool) string {
	var owner []int
	for w, wd := range vs.Words {
		for _, r := range wd.Text {
			if !unicode.IsSpace(r) {
				owner = append(owner, w)
			}
		}
	}
	var out []rune
	lastWord := -1 // the word of the last restored letter written, while it is the last rune written
	write := func(r rune, word int) {
		if len(out) > 0 && !unicode.IsSpace(r) && !unicode.IsSpace(out[len(out)-1]) && lastWord != word {
			if lastWord >= 0 || word >= 0 {
				out = append(out, ' ')
			}
		}
		out = append(out, r)
		lastWord = word
	}
	k := 0
	for _, df := range diff.DiffMatchPatch(vs.Text(), vs.ASRText) {
		for _, r := range df.Text {
			switch {
			case df.Type == diffmatchpatch.DiffInsert:
				write(r, -1)
			case unicode.IsSpace(r):
				if df.Type == diffmatchpatch.DiffEqual {
					write(r, -1)
				}
			default:
				if df.Type == diffmatchpatch.DiffEqual {
					write(r, -1)
				} else if k < len(owner) && restore[owner[k]] {
					write(r, owner[k])
				}
				k++
			}
		}
	}
	return strings.Join(strings.Fields(string(out)), " ")
}

// SelectMMSScores reads the fa_score that mms_align gave each word.
func SelectMMSScores(conn db.DBAdapter) (map[int64]float64, *log.Status) {
	scores := make(map[int64]float64)
	query := `SELECT word_id, fa_score FROM words WHERE ttype = 'W'`
	rows, err := conn.DB.Query(query)
	if err != nil {
		return scores, log.Error(conn.Ctx, 500, err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var wordId int64
		var score float64
		err = rows.Scan(&wordId, &score)
		if err != nil {
			return scores, log.Error(conn.Ctx, 500, err, query)
		}
		scores[wordId] = score
	}
	err = rows.Err()
	if err != nil {
		return scores, log.Error(conn.Ctx, 500, err, query)
	}
	return scores, nil
}

// FindMissingWords returns every whole word of verses missing from its transcript.  mmsScores holds
// the mms_align fa_score of each word, by word id.
func FindMissingWords(verses []generic.Verse2, mmsScores map[int64]float64) []MissingWord {
	var result []MissingWord
	for _, vs := range verses {
		if len(vs.Words) == 0 {
			continue
		}
		deleted := deletedLetters(vs.Text(), vs.ASRText)
		k := 0
		for w, wd := range vs.Words {
			n, d := 0, 0
			for _, r := range wd.Text {
				if unicode.IsSpace(r) {
					continue
				}
				if k < len(deleted) && deleted[k] {
					d++
				}
				n++
				k++
			}
			if n == 0 || d < n || hasDigit(wd.Text) {
				continue
			}
			mw := MissingWord{ScriptId: vs.ScriptId, LineRef: vs.LineRef, WordIndex: w, Text: wd.Text,
				QAScore: geoCharScore(wd), MMSScore: mmsScores[wd.WordId]}
			mw.Confirmed = mw.MMSScore < MMSLimit
			result = append(result, mw)
		}
	}
	return result
}

// deletedLetters returns, for each letter of ref, whether the character diff deletes it.
func deletedLetters(ref string, asr string) []bool {
	var result []bool
	for _, df := range diff.DiffMatchPatch(ref, asr) {
		if df.Type == diffmatchpatch.DiffInsert {
			continue
		}
		for _, r := range df.Text {
			if !unicode.IsSpace(r) {
				result = append(result, df.Type == diffmatchpatch.DiffDelete)
			}
		}
	}
	return result
}

func geoCharScore(wd generic.Word2) float64 {
	if len(wd.Chars) == 0 {
		return wd.FAScore
	}
	scores := make([]float64, len(wd.Chars))
	for i, ch := range wd.Chars {
		scores[i] = ch.FAScore
	}
	return fa.GeometricMean(scores)
}

func hasDigit(text string) bool {
	for _, r := range text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
