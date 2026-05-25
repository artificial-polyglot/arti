package asr_align

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
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

// This is solely for debugging
func selectWord(db *sql.DB, wordId int64) string {
	var word string
	err := db.QueryRow("SELECT word from words where word_id = ?", wordId).Scan(&word)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return word
		} else {
			panic(err)
		}
	}
	return word
}

// This is solely for debugging
func selectScript(db *sql.DB, scriptId int64) string {
	var text string
	err := db.QueryRow("SELECT script_text from scripts where script_id = ?", scriptId).Scan(&text)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return text
		} else {
			panic(err)
		}
	}
	return text
}
