package speech_to_text

import (
	"bytes"
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/lang_tree/search"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/*
Docs:
https://github.com/openai/whisper
Install:
pip3 install git+https://github.com/openai/whisper.git
Whisper is an open source Speech to Text program developed by OpenAI.
Executable:
/Users/gary/Library/Python/3.9/bin/whisper
*/

type Whisper struct {
	ctx     context.Context
	conn    db.DBAdapter
	bibleId string
	model   string
	lang2   string // 2 char language code
	tempDir string
}

func NewWhisper(bibleId string, conn db.DBAdapter, model string, lang2 string) Whisper {
	var w Whisper
	w.ctx = conn.Ctx
	w.conn = conn
	w.bibleId = bibleId
	w.model = model
	w.lang2 = lang2
	return w
}

func (w *Whisper) ProcessFiles(files []input.InputFile) *log.Status {
	var status *log.Status
	var outputFile string
	var err error
	w.tempDir, err = os.MkdirTemp(os.Getenv(`FCBH_DATASET_TMP`), "Whisper_")
	if err != nil {
		return log.Error(w.ctx, 500, err, `Error creating temp dir`)
	}
	defer os.RemoveAll(w.tempDir)
	if w.lang2 == `` {
		var tree = search.NewLanguageTree(w.ctx)
		err = tree.Load()
		if err != nil {
			return log.Error(w.ctx, 500, err, `Error loading language`)
		}
		langs, distance, err2 := tree.Search(strings.ToLower(w.bibleId[:3]), "whisper")
		if err2 != nil {
			return log.Error(w.ctx, 500, err2, `Error Searching for language`)
		}
		if len(langs) > 0 {
			w.lang2 = langs[0]
			log.Info(w.ctx, `Using language`, w.lang2, "distance:", distance)
		} else {
			return log.ErrorNoErr(w.ctx, 400, `No compatible language code was found for`, w.bibleId)
		}
	}
	for _, file := range files {
		fmt.Println(`INPUT FILE:`, file)
		var timestamps []db.Timestamp
		timestamps, status = w.conn.SelectScriptTimestamps(file.BookId, file.Chapter)
		if status != nil {
			return status
		}
		status = w.conn.DeleteScripts(file.BookId, file.Chapter)
		if status != nil {
			return status
		}
		var records []db.Script
		if len(timestamps) > 0 {
			timestamps, status = w.ChopByTimestamp(file, timestamps)
			if status != nil {
				return status
			}
			for pieceNum, piece := range timestamps {
				fmt.Println(`VERSE PIECE:`, piece)
				inputFile := filepath.Join(w.tempDir, piece.AudioFile)
				outputFile, status = w.RunWhisper(inputFile)
				var rec db.Script
				rec, status = w.loadWhisperVerses(outputFile, file, pieceNum, piece)
				records = append(records, rec)
			}
		} else {
			outputFile, status = w.RunWhisper(file.FilePath())
			records, status = w.loadWhisperOutput(outputFile, file)
		}
		w.conn.InsertScripts(records)
		records = nil
	}
	return status
}

func (w *Whisper) RunWhisper(audioFile string) (string, *log.Status) {
	var status *log.Status
	whisperPath := os.Getenv(`FCBH_WHISPER_EXE`)
	cmd := exec.Command(whisperPath,
		audioFile,
		`--model`, w.model,
		`--output_format`, `json`,
		`--fp16`, `False`,
		`--language`, w.lang2,
		`--word_timestamps`, `True`, // Runs about 10% faster with this off.  Should it be conditional?
		`--output_dir`, w.tempDir)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		status = log.Error(w.ctx, 500, err, stderrBuf.String())
		// Do not return immediately, must get std error
	}
	stderrStr := stderrBuf.String()
	if stderrStr != `` {
		log.Warn(w.ctx, `Whisper Stderr:`, stderrStr)
	}
	fileType := filepath.Ext(audioFile)
	filename := filepath.Base(audioFile)
	outputFile := filepath.Join(w.tempDir, filename[:len(filename)-len(fileType)]) + `.json`
	return outputFile, status
}
