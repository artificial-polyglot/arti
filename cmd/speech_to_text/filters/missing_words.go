package filters

import (
	"strconv"
	"unicode"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
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
// This filter reports, and does not change the verses.  Run it after FilterAlignmentErrors, so that
// words moved to another verse are not reported as missing.

// MMSLimit is the most mms_align fa_score of a missing word that is confirmed.  In N1SKNSEC, missing
// words scored either 0.175 or less, or 0.61 or more, and 11.9% of words that ASR heard score below it.
const MMSLimit = 0.2

// MissingWord is one word of the text that is wholly missing from the ASR transcript.
type MissingWord struct {
	ScriptId  int64
	LineRef   generic.VerseRef
	WordIndex int
	Text      string
	QAScore   float64 // mean qa_align score of the word's chars, from the trained model
	MMSScore  float64 // mms_align fa_score of the word, from the base model
	Confirmed bool    // both models find the word missing from the audio
}

func (m MissingWord) Reason() string {
	score := strconv.FormatFloat(m.MMSScore, 'f', 2, 64)
	if m.Confirmed {
		return "missing from audio: mms_align score " + score
	}
	return "doubtful: mms_align score " + score + " says the word may have been read"
}

// FilterMissingWords returns verses unchanged, and every whole word missing from a transcript, with
// the decision made about it.  verses must have qa_align chars and ASR transcripts, as
// proofing_rpt.SelectCharData and AddASRTranscript return them.
func FilterMissingWords(conn db.DBAdapter, verses []generic.Verse2) ([]generic.Verse2, []MissingWord, *log.Status) {
	scores, status := SelectMMSScores(conn)
	if status != nil {
		return verses, nil, status
	}
	return verses, FindMissingWords(verses, scores), nil
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
				QAScore: meanCharScore(wd), MMSScore: mmsScores[wd.WordId]}
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

func meanCharScore(wd generic.Word2) float64 {
	if len(wd.Chars) == 0 {
		return wd.FAScore
	}
	var total float64
	for _, ch := range wd.Chars {
		total += ch.FAScore
	}
	return total / float64(len(wd.Chars))
}

func hasDigit(text string) bool {
	for _, r := range text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
