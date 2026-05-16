package read

import (
	"context"
	"os"
	"strings"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/utility/safe"
)

type ScriptVersifier struct {
	ctx  context.Context
	conn db.DBAdapter
}

func NewScriptVersifier(conn db.DBAdapter) ScriptVersifier {
	var s ScriptVersifier
	s.ctx = conn.Ctx
	s.conn = conn
	return s
}

func (s ScriptVersifier) Process() (db.DBAdapter, *log.Status) {
	tmpDB, status := db.NewerDBAdapter(s.ctx, true, s.conn.User, s.conn.Project+"_tmp")
	if status != nil {
		return s.conn, status
	}
	chapters, status := s.conn.SelectBookChapter()
	if status != nil {
		return s.conn, status
	}
	for _, ch := range chapters {
		scripts, status1 := s.conn.SelectScriptsByChapter(ch.BookId, ch.ChapterNum)
		if status1 != nil {
			return s.conn, status1
		}
		var lines []string
		for _, scr := range scripts {
			if !strings.HasSuffix(scr.ScriptNum, "r") {
				lines = append(lines, scr.ScriptText)
			}
		}
		lines2 := safe.SafeStringJoin(lines)
		verses := s.ParseScriptToVerses(ch.BookId, ch.ChapterNum, lines2)
		//var vsMap = make(map[string][]string)
		//for _, vs := range verses {
		//	ans := vsMap[vs.VerseStr]
		//	ans = append(ans, vs.ScriptText)
		//}
		//for key, value := range vsMap {
		//	if len(value) > 1 {
		//		fmt.Println(key, len(value), value)
		//	}
		//}
		status = s.uniqueVerses(verses)
		if status != nil {
			return s.conn, status
		}
		status = tmpDB.InsertScripts(verses)
		if status != nil {
			return s.conn, status
		}
	}
	tmpDB.Close()
	s.conn.Close()
	dbPath := s.conn.DatabasePath
	err := os.Rename(tmpDB.DatabasePath, dbPath)
	if err != nil {
		return s.conn, log.Error(s.ctx, 500, err, "Error renaming tmp database")
	}
	s.conn = db.NewDBAdapter(s.ctx, dbPath) // reopen new database
	return s.conn, nil
}

func (s ScriptVersifier) ParseScriptToVerses(bookId string, chapter int, text string) []db.Script {
	const (
		begin = iota + 1
		inNum
		endNum
	)
	//var labels = []string{``, `BEGIN`, `INNUM`, `ENDNUM`}
	var results = make([]db.Script, 0, 64)
	var sumInput = len(text)
	var sumOutput = 0
	var verseNum = `0`
	var tmpNum []byte
	var index = 0
	var state = begin
	for index < len(text) {
		switch state {
		case begin:
			var part string
			search := text[index:]
			pos := strings.Index(search, `{`)
			if pos < 0 {
				part = search
				sumOutput += len(part)
			} else {
				part = search[:pos]
				state = inNum
				tmpNum = []byte{}
				sumOutput += len(part) + 1
			}
			verse := db.Script{
				BookId:      bookId,
				ChapterNum:  chapter,
				VerseStr:    verseNum,
				VerseNum:    safe.SafeVerseNum(verseNum),
				ScriptTexts: []string{strings.TrimSpace(part)},
			}
			results = append(results, verse)
			index += len(part) + 1
		case inNum:
			char := text[index]
			if char >= '0' && char <= '9' {
				tmpNum = append(tmpNum, char)
				index++
				sumOutput += 1
			} else if char == '}' {
				verseNum = string(tmpNum)
				state = endNum
				index++
				sumOutput += 1
			} else {
				start := max(0, index-50)
				end := min(len(text)-1, index+50)
				log.Warn(s.ctx, bookId, chapter, verseNum, `Invalid char in {nn, expect n or } found `,
					string(char), ` in `, text[start:end])
				verseNum = string(tmpNum)
				state = begin
			}
		case endNum:
			char := text[index]
			peek := text[index+1]
			if (char == '_' || char == '-') && peek == '{' {
				state = inNum
				tmpNum = []byte(verseNum + "-")
				index += 2
				sumOutput += 2
			} else {
				state = begin
			}
		}
	}
	if sumInput != sumOutput {
		log.Warn(s.ctx, "Bug: Not all data processed by consolidateScript input:", sumInput, " output:", sumOutput)
	}
	return results
}

func (s ScriptVersifier) uniqueVerses(records []db.Script) *log.Status {
	chars := []string{"", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	var uniqueVerse = make(map[string]bool)
	for r := range records {
		for i := 0; i < len(chars); i++ {
			verseStr := records[r].VerseStr + chars[i]
			_, found := uniqueVerse[verseStr]
			if !found {
				uniqueVerse[verseStr] = true
				if i > 0 {
					records[r].VerseStr = verseStr
				}
				break
			}
		}
	}
	return nil
}
