package proofing_rpt

import (
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
)

// SelectCharData reads the same rows as db.SelectFACharTimestamps, but builds
// the Verse2/Word2/Char2 hierarchy directly out of the result set as it scans,
// rather than materializing a flat slice first. Because the rows arrive
// ordered by word_id, seq, the grouping is done in a single pass: a change of
// script_id closes the current verse and a change of word_id closes the
// current word.
//
// The Silence, SilenceLong and IsASR fields of Char2 are not set here; they are
// computed by later passes over the data.
func SelectCharData(conn db.DBAdapter, cutoff float64) ([]Verse2, *log.Status) {
	var verses []Verse2
	var query = `SELECT s.script_id, s.book_id, s.chapter_num, s.verse_str,
				w.word_id, w.word, w.word_punct, c.seq, c.char, c.begin_ts, c.end_ts, c.fa_score
				FROM scripts s JOIN words w ON s.script_id = w.script_id
				LEFT OUTER JOIN chars_qa_align c ON w.word_id = c.word_id
				WHERE w.ttype = 'W' AND w.script_id IN (
       				SELECT DISTINCT w2.script_id
       				FROM words w2 JOIN words_qa_align q2 ON w2.word_id = q2.word_id
       				WHERE q2.fa_score <= ?)
				ORDER BY c.word_id, c.seq`
	rows, err := conn.DB.Query(query, cutoff)
	if err != nil {
		return verses, log.Error(conn.Ctx, 500, err, "Error during SelectCharData.")
	}
	defer func() {
		if err2 := rows.Close(); err2 != nil {
			log.Warn(conn.Ctx, `Error closing`, "SelectCharData stmt", err2)
		}
	}()

	var verse Verse2
	var words []Word2
	var word Word2
	var sumFAScore, cntFAScore float64
	haveVerse := false
	haveWord := false

	flushWord := func() {
		if !haveWord {
			return
		}
		if cntFAScore > 0 {
			word.FAScore = sumFAScore / cntFAScore
		}
		words = append(words, word)
	}
	flushVerse := func() {
		if !haveVerse {
			return
		}
		flushWord()
		verse.Words = words
		if len(words) > 0 {
			verse.BeginTS = words[0].BeginTS
			verse.EndTS = words[len(words)-1].EndTS
			verse.Duration = verse.EndTS - verse.BeginTS
		}
		verses = append(verses, verse)
	}

	for rows.Next() {
		var scriptId int64
		var lineRef generic.VerseRef
		var wordId int64
		var wordText, wordPunct string
		var charSeq int
		var chr string
		var beginTS, endTS, faScore float64
		err = rows.Scan(&scriptId, &lineRef.BookId, &lineRef.ChapterNum, &lineRef.VerseStr,
			&wordId, &wordText, &wordPunct, &charSeq, &chr, &beginTS, &endTS, &faScore)
		if err != nil {
			return verses, log.Error(conn.Ctx, 500, err, "Error in SelectCharData.")
		}
		if !haveVerse || scriptId != verse.ScriptId {
			flushVerse()
			verse = Verse2{ScriptId: scriptId, LineRef: lineRef}
			words = nil
			haveVerse = true
			haveWord = false // force a fresh word even if word_id repeats across the verse boundary
		}
		if !haveWord || wordId != word.WordId {
			flushWord()
			word = Word2{WordId: wordId, Ttype: "W", Text: wordText, WordPunct: wordPunct, BeginTS: beginTS}
			sumFAScore, cntFAScore = 0, 0
			haveWord = true
		}
		char, _ := utf8.DecodeRuneInString(chr)
		word.Chars = append(word.Chars, Char2{Char: char, BeginTS: beginTS, EndTS: endTS, FAScore: faScore})
		word.EndTS = endTS
		sumFAScore += faScore
		cntFAScore++
	}
	flushVerse()
	err = rows.Err()
	if err != nil {
		log.Warn(conn.Ctx, err, query)
	}
	return verses, nil
}
