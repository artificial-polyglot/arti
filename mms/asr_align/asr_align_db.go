package asr_align

import (
	"database/sql"
	"fmt"
	"strings"

	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

func (a *ASRAlign) selectVersesByBookChapter(bookId string, chapter int) ([]Char, *log.Status) {
	var results []Char
	var query = `SELECT s.script_id, w.word_id, s.verse_str, w.word_seq, w.word
	FROM scripts s JOIN words w ON w.script_id = s.script_id
	WHERE w.ttype IN ('W','S') AND s.book_id = ? AND s.chapter_num = ?
	ORDER BY s.script_id, w.word_id`
	rows, err := a.conn.DB.Query(query, bookId, chapter)
	if err != nil {
		return results, log.Error(a.ctx, 500, err, query, bookId, chapter)
	}
	defer rows.Close()
	for rows.Next() {
		var scriptId int64
		var wordId int64
		var verseStr string
		var wordSeq int
		var word string
		err = rows.Scan(&scriptId, &wordId, &verseStr, &wordSeq, &word)
		if err != nil {
			return results, log.Error(a.ctx, 500, err, query, bookId, chapter)
		}
		for _, ch := range strings.ToLower(word) {
			var char Char
			char.ScriptId = scriptId
			char.WordId = wordId
			char.VerseStr = verseStr
			char.WordSeq = wordSeq
			char.Char = ch
			results = append(results, char)
		}
	}
	err = rows.Err()
	if err != nil {
		return results, log.Error(a.ctx, 500, err, query, bookId, chapter)
	}
	return results, nil
}

func (a *ASRAlign) updateWords(db *sql.DB, words []Word) error {
	const query = `UPDATE words SET word_begin_ts = ?, word_end_ts = ? WHERE word_id = ?`
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	for _, w := range words {
		_, err := stmt.Exec(w.BeginTS, w.EndTS, w.WordId)
		if err != nil {
			return fmt.Errorf("failed to update word_id %d: %w", w.WordId, err)
		}
	}
	return tx.Commit()
}

func (a *ASRAlign) updateScripts(db *sql.DB, scripts []Script) error {
	const query = `UPDATE scripts SET script_begin_ts = ?, script_end_ts = ? WHERE script_id = ?`
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	for _, w := range scripts {
		_, err := stmt.Exec(w.BeginTS, w.EndTS, w.ScriptId)
		if err != nil {
			return fmt.Errorf("failed to update script_id %d: %w", w.ScriptId, err)
		}
	}
	return tx.Commit()
}
