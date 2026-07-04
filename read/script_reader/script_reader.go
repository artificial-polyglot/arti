package script_reader

import (
	"context"
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/safe"
)

// This program will read Excel data and load the audio_scripts table

type ScriptReader struct {
	ctx       context.Context
	db        db.DBAdapter
	testament request.Testament
	col       request.SheetColumns
}

func NewScriptReader(db db.DBAdapter, testament request.Testament,
	col request.SheetColumns) ScriptReader {
	var d ScriptReader
	d.ctx = db.Ctx
	d.db = db
	d.testament = testament
	d.col = col
	return d
}

func (r ScriptReader) ProcessFiles(files []input.InputFile) *log.Status {
	var status *log.Status
	for _, file := range files {
		status = r.Read(file.FilePath())
	}
	return status
}

func (r ScriptReader) Read(filePath string) *log.Status {
	var status *log.Status
	rows, err := ReadSheet(filePath)
	if err != nil {
		return log.Error(r.ctx, 500, err, "Error reading rows from Excel file:", filePath)
	}
	var uniqueRefs = make(map[string]bool)
	var records []db.Script
	if r.col.IsSet() {
		rows = r.findFirstLine(rows)
	}
	for i, row := range rows {
		if i == 0 {
			if !r.col.IsSet() {
				status = r.findColIndexes(row)
				if status != nil {
					return status
				}
				continue // skip the header row
			}
		}
		var rec db.Script
		switch row[r.col.BookCol] {
		case `JMS`:
			rec.BookId = `JAS`
		case `TTS`:
			rec.BookId = `TIT`
		case ``:
			return log.ErrorNoErr(r.ctx, 500, "Have", row[r.col.BookCol], "Missing book_id in Excel row", i+1, "of file:", filePath)
		default:
			rec.BookId = row[r.col.BookCol]
		}
		if r.testament.HasNT(rec.BookId) || r.testament.HasOT(rec.BookId) {
			chapter, err := strconv.ParseFloat(row[r.col.ChapterCol], 64)
			rec.ChapterNum = int(chapter)
			if err != nil {
				return log.Error(r.ctx, 500, err, "Error: chapter num is not numeric", row[r.col.ChapterCol], "Line:", i)
			}
			if r.col.VerseCol < 0 || row[r.col.VerseCol] == `<<` {
				rec.VerseStr = `0`
			} else {
				rec.VerseStr = row[r.col.VerseCol]
			}
			if strings.HasSuffix(rec.VerseStr, ".0") {
				rec.VerseStr = rec.VerseStr[:(len(rec.VerseStr) - 2)]
			}
			rec.VerseStr, status = r.uniqueVerse(uniqueRefs, rec)
			if status != nil {
				return status
			}
			rec.VerseNum = safe.SafeVerseNum(rec.VerseStr)
			if r.col.CharacterCol >= 0 {
				rec.Person = row[r.col.CharacterCol]
			}
			if r.col.ActorCol >= 0 {
				rec.Actor = row[r.col.ActorCol]
			}
			rec.ScriptNum = row[r.col.LineCol]
			if strings.HasSuffix(rec.ScriptNum, ".0") {
				rec.ScriptNum = rec.ScriptNum[:len(rec.ScriptNum)-2]
			}
			text := row[r.col.TextCol]
			//text = strings.Replace(text,'_x000D_','' ) // remove excel CR
			rec.ScriptTexts = append(rec.ScriptTexts, text)
			records = append(records, rec)
		}
	}
	status = r.db.InsertScripts(records)
	return status
}

func (r *ScriptReader) findColIndexes(heading []string) *log.Status {
	r.col.BookCol = -1
	r.col.ChapterCol = -1
	r.col.VerseCol = -1
	r.col.CharacterCol = -1
	r.col.ActorCol = -1
	r.col.LineCol = -1
	r.col.TextCol = -1
	for col, head := range heading {
		switch strings.ToLower(head) {
		case `book`, `bk`, `book name abbr`:
			r.col.BookCol = col
		case `chapter`, `cp`, `chapter number`:
			r.col.ChapterCol = col
		case `verse`, `verse_number`, `start verse number`:
			r.col.VerseCol = col
		case `line #`, `line_number`, `line id:`, `line`, `line number`:
			r.col.LineCol = col
		case `character`, `characters1`, `character group`:
			r.col.CharacterCol = col
		case `reader`, `reader name`:
			r.col.ActorCol = col
		case `target language`, `verse_content1`:
			r.col.TextCol = col
		}
	}
	var msgs []string
	if r.col.BookCol < 0 {
		msgs = append(msgs, `Book column was not found`)
	}
	if r.col.ChapterCol < 0 {
		msgs = append(msgs, `Chapter column was not found`)
	}
	if r.col.VerseCol < 0 {
		msgs = append(msgs, `Verse column was not found`)
	}
	if r.col.LineCol < 0 {
		msgs = append(msgs, `Line column was not found`)
	}
	if r.col.TextCol < 0 {
		msgs = append(msgs, `Text column was not found`)
	}
	var status *log.Status
	if len(msgs) > 0 {
		status = log.ErrorNoErr(r.ctx, 500, `Columns missing in script:`, strings.Join(msgs, `; `))
	}
	return status
}

func (r ScriptReader) uniqueVerse(uniqueRefs map[string]bool, rec db.Script) (string, *log.Status) {
	var verse string
	chars := []string{"", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	for i := 0; i < len(chars); i++ {
		verse = rec.VerseStr + chars[i]
		key := generic.VerseRef{
			BookId:     rec.BookId,
			ChapterNum: rec.ChapterNum,
			VerseStr:   verse}.UniqueKey()
		_, found := uniqueRefs[key]
		if !found {
			uniqueRefs[key] = true
			return verse, nil
		}
	}
	return verse, log.ErrorNoErr(r.ctx, 500, "Too many duplicate script lines for verse", rec.VerseStr,
		"in", rec.BookId, rec.ChapterNum)
}

// Find first line is used for xls files, or any file where the SheetColumns is used
func (r ScriptReader) findFirstLine(rows [][]string) [][]string {
	if r.col.BookCol == 0 {
		return rows
	}
	for i, row := range rows {
		maxChapter, found := db.BookChapterMap[row[r.col.BookCol]]
		if found {
			chapterNum, err := strconv.ParseFloat(row[r.col.ChapterCol], 64)
			if err == nil && int(chapterNum) <= maxChapter && chapterNum > 0.0 {
				return rows[i:]
			}
		}
	}
	return rows
}
