package proofing_rpt

import (
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
)

// SelectCharData reads data from scripts, words, chars, and chars_qa_align
// Note that fa_score at the character level is from chars_qa_align.  It is a result of the
// forced alignment done in the qa_align module.
func SelectCharData(conn db.DBAdapter, cutoff float64) ([]Verse2, *log.Status) {
	var verses []Verse2
	var query = `SELECT s.script_id, s.book_id, s.chapter_num, s.verse_str, s.script_begin_ts, s.script_end_ts,
				w.word_id, w.word, w.word_punct, w.word_begin_ts, w.word_end_ts, w.fa_score,
				qc.seq, qc.char, qc.begin_ts, qc.end_ts, qc.fa_score
				FROM scripts s JOIN words w ON s.script_id = w.script_id
				LEFT OUTER JOIN chars_qa_align qc ON w.word_id = qc.word_id
				WHERE w.ttype = 'W' AND w.script_id IN (
       				SELECT DISTINCT w2.script_id
       				FROM words w2 JOIN words_qa_align q2 ON w2.word_id = q2.word_id
       				WHERE q2.fa_score <= ?)
				ORDER BY qc.word_id, qc.seq`
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
	haveVerse := false
	haveWord := false

	flushWord := func() {
		if !haveWord {
			return
		}
		words = append(words, word)
	}
	flushVerse := func() {
		if !haveVerse {
			return
		}
		flushWord()
		verse.Words = words
		verses = append(verses, verse)
	}

	for rows.Next() {
		var scriptId int64
		var lineRef generic.VerseRef
		var wordId int64
		var wordText, wordPunct string
		var charSeq int
		var chr string
		var scriptBeginTS, scriptEndTS, wordBeginTS, wordEndTS, charBeginTS, charEndTS, wordFAScore, charFAScore float64
		err = rows.Scan(&scriptId, &lineRef.BookId, &lineRef.ChapterNum, &lineRef.VerseStr,
			&scriptBeginTS, &scriptEndTS,
			&wordId, &wordText, &wordPunct, &wordBeginTS, &wordEndTS, &wordFAScore,
			&charSeq, &chr, &charBeginTS, &charEndTS, &charFAScore)
		if err != nil {
			return verses, log.Error(conn.Ctx, 500, err, "Error in SelectCharData.")
		}
		if !haveVerse || scriptId != verse.ScriptId {
			flushVerse()
			verse = Verse2{ScriptId: scriptId, LineRef: lineRef, BeginTS: scriptBeginTS, EndTS: scriptEndTS}
			verse.Duration = scriptEndTS - scriptBeginTS
			words = nil
			haveVerse = true
			haveWord = false // force a fresh word even if word_id repeats across the verse boundary
		}
		if !haveWord || wordId != word.WordId {
			flushWord()
			word = Word2{WordId: wordId, Ttype: "W", Text: wordText, WordPunct: wordPunct, BeginTS: wordBeginTS,
				EndTS: wordEndTS, FAScore: wordFAScore}
			haveWord = true
		}
		char, _ := utf8.DecodeRuneInString(chr)
		word.Chars = append(word.Chars, Char2{Char: char, BeginTS: charBeginTS, EndTS: charEndTS, FAScore: charFAScore})
	}
	flushVerse()
	err = rows.Err()
	if err != nil {
		log.Warn(conn.Ctx, err, query)
	}
	return verses, nil
}
