package output

import (
	"database/sql"
	"encoding/json"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"strconv"
)

func (o *Output) LoadScriptStruct(d db.DBAdapter) ([]Script, *log.Status) {
	var results []Script
	var status *log.Status
	query := `SELECT scripts.script_id, book_id, chapter_num, chapter_end, audio_file, script_num,
		usfm_style, person, actor, verse_str, verse_end, script_text, 
		script_begin_ts, script_end_ts, rows, cols, mfcc_json
		FROM scripts LEFT OUTER JOIN script_mfcc ON script_mfcc.script_id = scripts.script_id
		ORDER BY scripts.script_id`
	rows, err := d.DB.Query(query)
	if err != nil {
		return results, log.Error(d.Ctx, 500, err, "Error during select scripts")
	}
	defer rows.Close()
	for rows.Next() {
		var sc Script
		var mfccRows sql.NullInt64
		var mfccCols sql.NullInt64
		var mfccJson sql.NullString
		err = rows.Scan(&sc.ScriptId, &sc.BookId, &sc.ChapterNum, &sc.ChapterEnd, &sc.AudioFile,
			&sc.ScriptNum, &sc.UsfmStyle, &sc.Person, &sc.Actor, &sc.VerseStr, &sc.VerseEnd,
			&sc.ScriptText, &sc.ScriptBeginTS, &sc.ScriptEndTS, &mfccRows, &mfccCols, &mfccJson)
		if err != nil {
			status = log.Error(d.Ctx, 500, err, "Error in Select Scripts.")
			return results, status
		}
		if mfccRows.Valid && mfccCols.Valid && mfccJson.Valid { //&& len(mfccJson.String) > 0 {
			sc.MFCCRows = int(mfccRows.Int64)
			sc.MFCCCols = int(mfccCols.Int64)
			err = json.Unmarshal([]byte(mfccJson.String), &sc.MFCC)
			if err != nil {
				status = log.Error(d.Ctx, 500, err, "Error in Unmarshalling MFCC")
				return results, status
			}
		}
		sc.Reference = o.FormatReference(sc.BookId, sc.ChapterNum, sc.ChapterEnd, sc.VerseStr, sc.VerseEnd)
		results = append(results, sc)
	}
	err = rows.Err()
	if err != nil {
		log.Warn(d.Ctx, err, query)
	}
	return results, status
}

func (o *Output) LoadWordStruct(d db.DBAdapter) ([]Word, *log.Status) {
	var results []Word
	var status *log.Status
	query := `SELECT words.word_id, words.script_id, book_id, chapter_num, chapter_end, verse_str,
		verse_end, words.verse_num, usfm_style, person, actor, 
		word_seq, word, word_begin_ts, word_end_ts, word_enc, rows, cols, mfcc_json
		FROM words JOIN scripts ON words.script_id = scripts.script_id
		LEFT OUTER JOIN word_mfcc ON word_mfcc.word_id = words.word_id
		WHERE ttype = 'W'
		ORDER BY words.word_id`
	rows, err := d.DB.Query(query)
	if err != nil {
		status = log.Error(d.Ctx, 500, err, "Error during select words")
		return results, status
	}
	defer rows.Close()
	for rows.Next() {
		var wd Word
		var mfccRows sql.NullInt64
		var mfccCols sql.NullInt64
		var mfccJson sql.NullString
		var wordJson string
		err = rows.Scan(&wd.WordId, &wd.ScriptId, &wd.BookId, &wd.ChapterNum, &wd.ChapterEnd,
			&wd.VerseStr, &wd.VerseEnd, &wd.VerseNum, &wd.UsfmStyle, &wd.Person, &wd.Actor,
			&wd.WordSeq, &wd.Word, &wd.WordBeginTS, &wd.WordEndTS,
			&wordJson, &mfccRows, &mfccCols, &mfccJson)
		if err != nil {
			status = log.Error(d.Ctx, 500, err, "Error in Select Words.")
			return results, status
		}
		if mfccRows.Valid && mfccCols.Valid && mfccJson.Valid {
			wd.MFCCRows = int(mfccRows.Int64)
			wd.MFCCCols = int(mfccCols.Int64)
			err = json.Unmarshal([]byte(mfccJson.String), &wd.MFCC)
			if err != nil {
				status = log.Error(d.Ctx, 500, err, "Error in Unmarshalling MFCC")
				return results, status
			}
		}
		if len(wordJson) > 0 {
			err = json.Unmarshal([]byte(wordJson), &wd.WordEnc)
			if err != nil {
				status = log.Error(d.Ctx, 500, err, "Error in Unmarshalling WordEnc")
				return results, status
			}
		}
		wd.Reference = o.FormatReference(wd.BookId, wd.ChapterNum, wd.ChapterEnd, wd.VerseStr, wd.VerseEnd)
		results = append(results, wd)
	}
	err = rows.Err()
	if err != nil {
		log.Warn(d.Ctx, err, query)
	}
	return results, status
}

func (o *Output) FormatReference(bookId string, chapterNum int, chapterEnd int, verseStr string, verseEnd string) string {
	var result = bookId + ` ` + strconv.Itoa(chapterNum) + `:` + verseStr
	if chapterEnd != 0 && chapterNum != chapterEnd {
		result += `-` + strconv.Itoa(chapterEnd) + `:` + verseEnd
	} else if verseEnd != `` && verseStr != verseEnd {
		result += `-` + verseEnd
	}
	return result
}
