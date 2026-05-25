package speech_to_text

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/safe"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func (w *Whisper) ChopByTimestamp(file input.InputFile, timestamps []db.Timestamp) ([]db.Timestamp, *log.Status) {
	var results []db.Timestamp
	var status *log.Status
	var command []string
	command = append(command, `-i`, file.FilePath())
	command = append(command, `-codec:a`, `copy`)
	command = append(command, `-y`)
	for _, ts := range timestamps {
		if ts.BeginTS == 0.0 && ts.EndTS == 0.0 {
			continue
		}
		beginTS := strconv.FormatFloat(ts.BeginTS, 'f', 2, 64)
		command = append(command, `-ss`, beginTS)
		if ts.EndTS != 0.0 {
			endTS := strconv.FormatFloat(ts.EndTS, 'f', 2, 64)
			command = append(command, `-to`, endTS)
		}
		ts.AudioFile = fmt.Sprintf("verse_%s_%d_%s_%s.mp3",
			file.BookId, file.Chapter, ts.VerseStr, beginTS)
		outputPath := filepath.Join(w.tempDir, ts.AudioFile)
		command = append(command, `-c`, `copy`, outputPath)
		results = append(results, ts)
	}
	ffMpegPath := `ffmpeg`
	cmd := exec.Command(ffMpegPath, command...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		status = log.Error(w.ctx, 500, err, stderrBuf.String())
	}
	return results, status
}

type WhisperWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Prob  float64 `json:"probability"`
}

type WhisperSegmentType struct {
	Id               int           `json:"id"`
	Seek             float64       `json:"seek"`
	Start            float64       `json:"start"`
	End              float64       `json:"end"`
	Text             string        `json:"text"`
	Tokens           []int         `json:"tokens"`
	Temperature      float32       `json:"temperature"`
	AvgLogProb       float64       `json:"avg_logprob"`
	CompressionRatio float64       `json:"compression_ratio"`
	NoSpeechProb     float64       `json:"no_speech_prob"`
	Words            []WhisperWord `json:"words"`
}
type WhisperOutputType struct {
	Segments []WhisperSegmentType `json:"segments"`
	Language string               `json:"language"`
}

func (w *Whisper) loadWhisperOutput(outputFile string, file input.InputFile) ([]db.Script, *log.Status) {
	var status *log.Status
	var records = make([]db.Script, 0, 100)
	content, err := os.ReadFile(outputFile)
	if err != nil {
		status = log.Error(w.ctx, 500, err, `Error reading file`)
		return records, status
	}
	var response WhisperOutputType
	err = json.Unmarshal(content, &response)
	if err != nil {
		status = log.Error(w.ctx, 500, err, "Error decoding Whisper JSON")
		return records, status
	}
	for i, seg := range response.Segments {
		var rec db.Script
		rec.BookId = file.BookId
		rec.ChapterNum = file.Chapter
		rec.AudioFile = file.Filename
		rec.ScriptNum = strconv.Itoa(i + 1) // Works because it process 1 chapter per file.
		rec.VerseNum = 0
		rec.VerseStr = ``
		rec.ScriptTexts = []string{seg.Text}
		rec.ScriptBeginTS = seg.Start
		rec.ScriptEndTS = seg.End
		records = append(records, rec)
	}
	return records, status
}

func (w *Whisper) loadWhisperVerses(outputFile string, file input.InputFile,
	pieceNum int, piece db.Timestamp) (db.Script, *log.Status) {
	var rec db.Script
	var status *log.Status
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return rec, log.Error(w.ctx, 500, err, `Error reading file`)
	}
	var response WhisperOutputType
	err = json.Unmarshal(content, &response)
	if err != nil {
		return rec, log.Error(w.ctx, 500, err, "Error decoding Whisper JSON")
	}
	rec.BookId = file.BookId
	rec.ChapterNum = file.Chapter
	rec.AudioFile = file.Filename
	rec.ScriptNum = strconv.Itoa(pieceNum + 1)
	rec.VerseNum = safe.SafeVerseNum(piece.VerseStr)
	rec.VerseStr = piece.VerseStr
	for i, seg := range response.Segments {
		rec.ScriptTexts = append(rec.ScriptTexts, seg.Text)
		if i == 0 {
			rec.ScriptBeginTS = seg.Start + piece.BeginTS
		}
		rec.ScriptEndTS = seg.End + piece.BeginTS
	}
	return rec, status
}
