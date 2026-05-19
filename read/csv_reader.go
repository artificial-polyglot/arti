package read

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

// This program will read Excel data and load the audio_scripts table

type CSVReader struct {
	ctx context.Context
	db  db.DBAdapter
}

func NewCSVReader(db db.DBAdapter) CSVReader {
	var d CSVReader
	d.ctx = db.Ctx
	d.db = db
	return d
}

func (r CSVReader) ProcessFiles(files []input.InputFile) *log.Status {
	var status *log.Status
	for _, file := range files {
		status = r.Read(file.FilePath())
	}
	return status
}

func (r CSVReader) Read(filePath string) *log.Status {
	var status *log.Status
	file, err := os.Open(filePath)
	if err != nil {
		return log.Error(r.ctx, 500, err, "Error: could not open", filePath)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	var records []db.Script
	var first = true
	var col CSVIndex
	var row []string
	for {
		row, err = reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return log.Error(r.ctx, 500, err, "Error: could not read", filePath)
		}
		if first {
			first = false
			col, status = r.FindColIndexes(row)
			if status != nil {
				return status
			}
		} else {
			var rec db.Script
			switch row[col.BookCol] {
			case `JMS`:
				rec.BookId = `JAS`
			case `TTS`:
				rec.BookId = `TIT`
			case ``:
				return log.ErrorNoErr(r.ctx, 500, `Error: Did not find book_id`)
			default:
				rec.BookId = row[col.BookCol]
			}
			rec.ChapterNum, err = strconv.Atoi(row[col.ChapterCol])
			if err != nil {
				return log.Error(r.ctx, 500, err, "Error: chapter num is not numeric", row[col.ChapterCol])
			}
			if col.VerseCol >= 0 {
				rec.VerseStr = row[col.VerseCol]
				rec.VerseNum, err = strconv.Atoi(row[col.VerseCol])
				if err != nil {
					return log.Error(r.ctx, 500, err, "Error: verse number is not numeric in file", filePath,
						"value:", row[col.VerseCol])
				}
			}
			rec.ScriptNum = row[col.LineCol]
			text := row[col.TextCol]
			rec.ScriptTexts = append(rec.ScriptTexts, text)
			//if rec.ScriptNum[len(rec.ScriptNum)-1] != 'r' {
			records = append(records, rec)
			//}
		}
	}
	status = r.db.InsertScripts(records)
	return status
}

type CSVIndex struct {
	BookCol    int
	ChapterCol int
	VerseCol   int
	LineCol    int
	TextCol    int
}

func (r CSVReader) FindColIndexes(heading []string) (CSVIndex, *log.Status) {
	var c CSVIndex
	c.BookCol = -1
	c.ChapterCol = -1
	c.VerseCol = -1
	c.LineCol = -1
	c.TextCol = -1
	for col, head := range heading {
		switch strings.ToLower(head) {
		case `book_id`:
			c.BookCol = col
		case `chapter`:
			c.ChapterCol = col
		case `verse`:
			c.VerseCol = col
		case `line_number`, `line id:`:
			c.LineCol = col
		case `text`, `transcribed_text`:
			c.TextCol = col
		}
	}
	var msgs []string
	if c.BookCol < 0 {
		msgs = append(msgs, `Book column was not found`)
	}
	if c.ChapterCol < 0 {
		msgs = append(msgs, `Chapter column was not found`)
	}
	if c.LineCol < 0 {
		msgs = append(msgs, `Line column was not found`)
	}
	if c.TextCol < 0 {
		msgs = append(msgs, `Text column was not found`)
	}
	if len(msgs) > 0 {
		return c, log.ErrorNoErr(r.ctx, 500, `Columns missing in csv file`, strings.Join(msgs, `; `))
	}
	return c, nil
}
