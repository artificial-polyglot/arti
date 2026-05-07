package asr_align

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/mms"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/utility/ffmpeg"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/utility/stdio_exec"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Char struct {
	BookId   string
	Chapter  int
	VerseStr string
	WordSeq  int  // questionable need
	CharSeq  int  // questionable need
	Char     rune `json:"ch"`
	StartTS  float64
	EndTS    float64 `json:"ts"`
}

type ASRAlign struct {
	ctx          context.Context
	conn         db.DBAdapter
	lang         string
	sttLang      string
	adapter      bool
	mmsAsrPy     *stdio_exec.StdioExec
	diffMatch    *diffmatchpatch.DiffMatchPatch
	versePattern *regexp.Regexp
	testing      bool // set in asr_align_test.go
}

func NewASRAlign(ctx context.Context, conn db.DBAdapter, lang string, sttLang string, adapter bool) ASRAlign {
	var a ASRAlign
	a.ctx = ctx
	a.conn = conn
	a.lang = lang
	a.sttLang = sttLang
	a.adapter = adapter
	a.diffMatch = diffmatchpatch.New()
	a.versePattern = regexp.MustCompile(`\{(\d+)\}`)
	return a
}

func (a *ASRAlign) ProcessFiles(files []input.InputFile) *log.Status {
	var status *log.Status
	tempDir, err := os.MkdirTemp(os.Getenv(`FCBH_DATASET_TMP`), "mms_asr_align_")
	if err != nil {
		return log.Error(a.ctx, 500, err, `Error creating temp dir`)
	}
	defer os.RemoveAll(tempDir)
	var lang = a.lang
	if a.sttLang != "" {
		lang = a.sttLang
	}
	if !a.adapter {
		lang, status = mms.CheckLanguage(a.ctx, a.lang, a.sttLang, "mms_asr")
		if status != nil {
			return status
		}
	}
	status = a.conn.UpdateASRLanguage(lang)
	if status != nil {
		return status
	}
	pythonScript := filepath.Join(os.Getenv("GOPROJ"), "mms/asr_align/asr_align.py")
	var useAdapter string
	if a.adapter {
		useAdapter = "adapter"
	}
	a.mmsAsrPy, status = stdio_exec.NewStdioExec(a.ctx, os.Getenv(`FCBH_MMS_ASR_PYTHON`), pythonScript, lang, useAdapter)
	if status != nil {
		return status
	}
	defer a.mmsAsrPy.Close()
	for _, file := range files {
		status = a.processFile(file, tempDir)
		if status != nil {
			return status
		}
	}
	return status
}

func (a *ASRAlign) processFile(file input.InputFile, tempDir string) *log.Status {
	response, status := a.processASR(file, tempDir)
	if status != nil {
		return status
	}
	var audioChars []Char
	err := json.Unmarshal([]byte(response), &audioChars)
	if err != nil {
		return log.Error(a.ctx, 500, err, "Error Unmarshalling ASR Response")
	}
	for i, c := range audioChars {
		audioChars[i].BookId = file.BookId
		audioChars[i].Chapter = file.Chapter
		print(string(c.Char))
	}
	textChars, status1 := a.selectVersesByBookChapter(file.BookId, file.Chapter)
	if status1 != nil {
		return status1
	}
	mergeChars := a.merge(audioChars, textChars)
	fmt.Println(mergeChars)
	return status
}

func (a *ASRAlign) processASR(file input.InputFile, tempDir string) (string, *log.Status) {
	var response string
	testFile := file.MediaId + "_" + file.BookId + "_" + strconv.Itoa(file.Chapter) + ".txt"
	_, err := os.Stat(testFile)
	if a.testing && !os.IsNotExist(err) {
		bytes, err2 := os.ReadFile(testFile)
		if err2 != nil {
			return response, log.Error(a.ctx, 500, err2, "Error Reading Test File.")
		}
		response = string(bytes)
	} else {
		wavFile, status := ffmpeg.ConvertMp3ToWav(a.ctx, tempDir, file.FilePath())
		if status != nil {
			return response, status
		}
		response, status = a.mmsAsrPy.Process(wavFile)
		if status != nil {
			return response, status
		}
		if a.testing {
			err = os.WriteFile(testFile, []byte(response), 0644)
			if err != nil {
				return response, log.Error(a.ctx, 500, err, "Error Writing Test File.")
			}
		}
	}
	return response, nil
}

func (a *ASRAlign) selectVersesByBookChapter(bookId string, chapter int) ([]Char, *log.Status) {
	var results []Char
	var query = `SELECT s.script_id, s.verse_str, LOWER(GROUP_CONCAT(w.word, ' ')) AS text
	FROM scripts s JOIN words w ON w.script_id = s.script_id
	WHERE w.ttype = 'W' AND s.book_id = ? AND s.chapter_num = ?
	GROUP BY s.script_id, s.verse_str
	ORDER BY s.script_id, s.verse_str`
	rows, err := a.conn.DB.Query(query, bookId, chapter)
	if err != nil {
		return results, log.Error(a.ctx, 500, err, query, bookId, chapter)
	}
	defer rows.Close()
	for rows.Next() {
		var scriptId int64
		var verseStr string
		var text string
		err = rows.Scan(&scriptId, &verseStr, &text)
		if err != nil {
			return results, log.Error(a.ctx, 500, err, query, bookId, chapter)
		}
		for _, ch := range text {
			var char Char
			char.BookId = bookId
			char.Chapter = chapter
			char.VerseStr = verseStr
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

type RuneDiff struct {
	Type diffmatchpatch.Operation
	Rune rune
}

/*
*
Compare the two []Char.  Output a []Char that is assigned VerseStr, and include
*/
func (a *ASRAlign) merge(audioChars []Char, textChars []Char) []Char {
	var results []Char
	var audioStr = a.convertChar2String(audioChars)
	var textStr = a.convertChar2String(textChars)
	lenAudio := utf8.RuneCountInString(audioStr)
	lenText := utf8.RuneCountInString(textStr)
	fmt.Println(lenAudio, lenText)
	diffs := a.diffMatch.DiffMain(audioStr, textStr, false)
	var runeDiffs []RuneDiff
	for _, diff := range diffs {
		for _, r := range diff.Text {
			runeDiffs = append(runeDiffs, RuneDiff{Type: diff.Type, Rune: r})
		}
	}
	var audioIndex = 0
	var textIndex = 0
	for _, diff := range runeDiffs {
		var selected Char
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			selected = audioChars[audioIndex]
			selected.VerseStr = textChars[textIndex].VerseStr
			audioIndex++
			textIndex++
		case diffmatchpatch.DiffDelete:
			selected = audioChars[audioIndex]
			if textIndex < len(textChars) {
				selected.VerseStr = textChars[textIndex].VerseStr
			}
			audioIndex++
		case diffmatchpatch.DiffInsert:
			selected = textChars[textIndex]
			textIndex++
		}
		results = append(results, selected)
	}
	return results
}

func (a *ASRAlign) convertChar2String(chars []Char) string {
	var result []rune
	for _, ch := range chars {
		result = append(result, ch.Char)
	}
	return string(result)
}

/*
func (a *ASRAlign) ensureASRTable() *log.Status {
	query := `CREATE TABLE IF NOT EXISTS asr (
		script_id INTEGER PRIMARY KEY,
		script_text TEXT NOT NULL,
		uroman TEXT NOT NULL DEFAULT '')`
	_, err := a.conn.DB.Exec(query)
	if err != nil {
		return log.Error(a.ctx, 500, err, query)
	}
	return nil
}

func (a *ASRAlign) insertASRText(scripts []asrScript) *log.Status {
	_, err := a.conn.DB.Exec(`DELETE FROM asr`)
	if err != nil {
		return log.Error(a.ctx, 500, err, "could not delete asr")
	}
	query := `INSERT INTO asr (script_id, script_text, uroman) VALUES (?,?,?)`
	tx, err := a.conn.DB.Begin()
	if err != nil {
		return log.Error(a.ctx, 500, err, query)
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		return log.Error(a.ctx, 500, err, query)
	}
	defer stmt.Close()
	for _, rec := range scripts {
		_, err = stmt.Exec(rec.scriptId, rec.text, rec.uRoman)
		if err != nil {
			return log.Error(a.ctx, 500, err, `Error while inserting asr text.`)
		}
	}
	err = tx.Commit()
	if err != nil {
		return log.Error(a.ctx, 500, err, query)
	}
	return nil
}
*/
