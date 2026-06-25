package qa_align

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/mms"
	"github.com/artificial-polyglot/arti/utility/ffmpeg"
	"github.com/artificial-polyglot/arti/utility/stdio_exec"
)

type FARequest struct {
	ScriptId      int64  `json:"script_id"`
	AudioPath     string `json:"audio"`
	ReferenceText string `json:"text"`
}

type QAAlign struct {
	ctx      context.Context
	conn     db.DBAdapter
	lang     string
	sttLang  string
	adapter  bool
	mmsAsrPy *stdio_exec.StdioExec
}

func NewQAAlign(ctx context.Context, conn db.DBAdapter, lang string, sttLang string, adapter bool) QAAlign {
	var a QAAlign
	a.ctx = ctx
	a.conn = conn
	a.lang = lang
	a.sttLang = sttLang
	a.adapter = adapter
	return a
}

// ProcessFiles will perform Auto Speech Recognition on these files
func (a *QAAlign) ProcessFiles() *log.Status {
	var status *log.Status
	tempDir, err := os.MkdirTemp(os.Getenv(`FCBH_DATASET_TMP`), "qa_align_")
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
	pythonScript := filepath.Join(os.Getenv("GOPROJ"), "mms/qa_align/qa_align.py")
	var useAdapter string
	if a.adapter {
		useAdapter = "adapter"
	}
	a.mmsAsrPy, status = stdio_exec.NewStdioExec(a.ctx, os.Getenv(`FCBH_MMS_ASR_PYTHON`), pythonScript, lang, useAdapter)
	if status != nil {
		return status
	}
	defer a.mmsAsrPy.Close()

	if err = createQAAlignTables(a.conn.DB); err != nil {
		return log.Error(a.ctx, 500, err, "Create table error")
	}
	var files []input.InputFile
	files, status = input.SelectAudioFiles(a.conn)
	for _, file := range files {
		status = a.processFile(file, tempDir)
		if status != nil {
			return status
		}
	}
	return status
}

func (a *QAAlign) processFile(file input.InputFile, tempDir string) *log.Status {
	var status *log.Status
	wavFile, status := ffmpeg.ConvertMp3ToWav(a.ctx, tempDir, file.FilePath())
	if status != nil {
		return status
	}
	var audioFiles []db.Audio
	if file.ScriptLine != "" { // This is dangerously dependant upon code elsewhere to set this under right conditions
		var audioFile db.Audio
		audioFile, status = a.selectScriptLine(file.ScriptLine)
		if status != nil {
			return status
		}
		if audioFile.ScriptEndTS == 0.0 {
			return nil
		}
		log.Info(a.ctx, "MMS ASR", audioFile.BookId, audioFile.ChapterNum, file.ScriptLine)
		audioFile.AudioVerseWav = wavFile
		audioFiles = append(audioFiles, audioFile)
	} else {
		log.Info(a.ctx, "MMS ASR", file.BookId, file.Chapter)
		audioFiles, status = a.conn.SelectFAScriptTimestamps(file.BookId, file.Chapter)
		if status != nil {
			return status
		}
		audioFiles, status = ffmpeg.ChopByTimestamp(a.ctx, tempDir, wavFile, audioFiles)
	}
	for i, ts := range audioFiles {
		fmt.Println(ts.BookId, ts.ChapterNum, ts.VerseStr, "sid:", ts.ScriptId)
		var request FARequest
		request.ScriptId = ts.ScriptId
		request.AudioPath = ts.AudioVerseWav
		request.ReferenceText = ts.Text
		content, err := json.Marshal(request)
		fmt.Println(request, ts.BeginTS, ts.EndTS)
		response, status1 := a.mmsAsrPy.Process(string(content))
		if status1 != nil {
			return status1
		}
		fmt.Println("response:", response)
		err = ProcessFAResults(a.conn.DB, request, response)
		if err != nil {
			return log.Error(a.ctx, 500, err, "Error Processing scriptId:", audioFiles[i].ScriptId)
		}
	}
	return nil
}

func (a *QAAlign) selectScriptLine(scriptLine string) (db.Audio, *log.Status) {
	var rec db.Audio
	var query = `SELECT script_id, book_id, chapter_num, audio_file, script_begin_ts, script_end_ts FROM scripts WHERE script_num = ?`
	row := a.conn.DB.QueryRow(query, scriptLine)
	err := row.Scan(&rec.ScriptId, &rec.BookId, &rec.ChapterNum, &rec.AudioFile, &rec.ScriptBeginTS, &rec.ScriptEndTS)
	if err != nil {
		return rec, log.Error(a.ctx, 500, err, "Error during SelectScriptLine.")
	}
	return rec, nil
}
