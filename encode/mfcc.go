package encode

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type MFCC struct {
	ctx     context.Context
	conn    db.DBAdapter
	bibleId string
	detail  request.Detail
	numMFCC int
}

func NewMFCC(ctx context.Context, conn db.DBAdapter, bibleId string,
	detail request.Detail, numMFCC int) MFCC {
	var m MFCC
	m.ctx = ctx
	m.conn = conn
	m.bibleId = bibleId
	m.detail = detail
	m.numMFCC = numMFCC
	return m
}

func (m *MFCC) ProcessFiles(audioFiles []input.InputFile) *log.Status {
	var status *log.Status
	for _, aFile := range audioFiles {
		var mfccResp MFCCResp
		mfccResp, status = m.executeLibrosa(aFile.FilePath())
		if status != nil {
			return status
		}
		if m.detail.Lines {
			status = m.processScripts(mfccResp, aFile.BookId, aFile.Chapter)
			if status != nil {
				return status
			}
		}
		if m.detail.Words {
			status = m.processWords(mfccResp, aFile.BookId, aFile.Chapter)
			if status != nil {
				return status
			}
		}
	}
	return status
}

type MFCCResp struct {
	AudioFile  string      `json:"input_file"`
	SampleRate float64     `json:"sample_rate"`
	HopLength  float64     `json:"hop_length"`
	FrameRate  float64     `json:"frame_rate"`
	Shape      []int       `json:"mfcc_shape"`
	Type       string      `json:"mfcc_type"`
	MFCC       [][]float32 `json:"mfccs"`
}

func (m *MFCC) executeLibrosa(audioFile string) (MFCCResp, *log.Status) {
	var result MFCCResp
	pythonPath := os.Getenv(`FCBH_LIBROSA_PYTHON`)
	mfccLibrosaPath := filepath.Join(os.Getenv(`GOPROJ`), `encode`, `mfcc_librosa.py`)
	cmd := exec.Command(pythonPath, mfccLibrosaPath, audioFile, strconv.Itoa(m.numMFCC))
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		_ = log.Error(m.ctx, 500, err, `Error executing mfcc_librosa.py`)
		// Don't return here, need to see stderr
	}
	if stderrBuf.Len() > 0 {
		return result, log.ErrorNoErr(m.ctx, 500, `mfcc_librosa.py stderr:`, stderrBuf.String())
	}
	if stdoutBuf.Len() == 0 {
		return result, log.ErrorNoErr(m.ctx, 500, `mfcc_librosa.py has no output.`)
	}
	err = json.Unmarshal(stdoutBuf.Bytes(), &result)
	if err != nil {
		return result, log.Error(m.ctx, 500, err, `Error parsing json from librosa`)
	}
	return result, nil
}

func (m *MFCC) processScripts(mfcc MFCCResp, bookId string, chapterNum int) *log.Status {
	var status *log.Status
	timestamps, status := m.conn.SelectScriptTimestamps(bookId, chapterNum)
	if status != nil {
		return status
	}
	mfccs := m.segmentMFCC(timestamps, mfcc)
	status = m.conn.InsertScriptMFCCS(mfccs)
	return status
}

func (m *MFCC) processWords(mfcc MFCCResp, bookId string, chapterNum int) *log.Status {
	var status *log.Status
	timestamps, status := m.conn.SelectWordTimestamps(bookId, chapterNum)
	mfccs := m.segmentMFCC(timestamps, mfcc)
	status = m.conn.InsertWordMFCCS(mfccs)
	return status
}

func (m *MFCC) segmentMFCC(timestamps []db.Timestamp, mfcc MFCCResp) []db.MFCC {
	var result []db.MFCC
	for _, ts := range timestamps {
		startIndex := int(ts.BeginTS*mfcc.FrameRate + 0.5)
		endIndex := int(ts.EndTS*mfcc.FrameRate + 0.5)
		var segment [][]float32
		if endIndex != 0 {
			segment = mfcc.MFCC[startIndex:endIndex][:]
		} else {
			segment = mfcc.MFCC[startIndex:][:] // The last timestamp from DB will be zero
		}
		var mf db.MFCC
		mf.Id = ts.Id
		mf.Rows = len(segment)
		if mf.Rows == 0 {
			mf.Cols = 0
		} else {
			mf.Cols = len(segment[0])
		}
		mf.MFCC = segment
		result = append(result, mf)
	}
	return result
}
