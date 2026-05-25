package read

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
)

type DBPTextReader struct {
	ctx       context.Context
	conn      db.DBAdapter
	testament request.Testament
}

func NewDBPTextReader(conn db.DBAdapter, testament request.Testament) DBPTextReader {
	var d DBPTextReader
	d.ctx = conn.Ctx
	d.conn = conn
	d.testament = testament
	return d
}

func (d *DBPTextReader) ProcessFiles(files []input.InputFile) *log.Status {
	var status *log.Status
	for _, file := range files {
		filePath := filepath.Join(file.Directory, file.Filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return log.Error(d.ctx, 500, err, "Error reading file:", filePath)
		}
		var scriptNum = 0
		var lastBookId string
		type TempRec struct {
			BookId     string `json:"book_id"`
			BookSeq    int
			ChapterNum int    `json:"chapter"`
			VerseStart int    `json:"verse_start"`
			VerseEnd   int    `json:"verse_end"`
			Text       string `json:"verse_text"`
		}
		type TempResp struct {
			Data []TempRec `json:"data"`
		}
		var response TempResp
		err = json.Unmarshal(content, &response)
		if err != nil {
			return log.Error(d.ctx, 500, err, "Error parsing JSON in plain_text file:", filePath)
		}
		var verses = response.Data
		for i, vs := range verses {
			verses[i].BookSeq = db.BookSeqMap[vs.BookId]
		}
		sort.Slice(verses, func(i, j int) bool {
			if verses[i].BookSeq != verses[j].BookSeq {
				return verses[i].BookSeq < verses[j].BookSeq
			}
			if verses[i].ChapterNum != verses[j].ChapterNum {
				return verses[i].ChapterNum < verses[j].ChapterNum
			}
			return verses[i].VerseStart < verses[j].VerseStart
		})
		var records = make([]db.Script, 0, 1000)
		for _, vs := range verses {
			if d.testament.HasNT(vs.BookId) || d.testament.HasOT(vs.BookId) {
				scriptNum++
				if vs.BookId != lastBookId {
					lastBookId = vs.BookId
					scriptNum = 1
				}
				var rec db.Script
				rec.ScriptNum = strconv.Itoa(scriptNum)
				rec.BookId = vs.BookId
				rec.ChapterNum = vs.ChapterNum
				rec.VerseNum = vs.VerseStart
				if vs.VerseStart == vs.VerseEnd {
					rec.VerseStr = strconv.Itoa(vs.VerseStart)
				} else {
					rec.VerseStr = strconv.Itoa(vs.VerseStart) + `-` + strconv.Itoa(vs.VerseEnd)
				}
				text := strings.Replace(vs.Text, "&lt", "<", -1)
				text = strings.Replace(text, "&gt", ">", -1)
				rec.ScriptTexts = append(rec.ScriptTexts, text)
				records = append(records, rec)
			}
		}
		status = d.conn.InsertScripts(records)
	}
	return status
}
