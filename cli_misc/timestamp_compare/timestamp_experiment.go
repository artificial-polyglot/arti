package main

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/cli_misc"
	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/fetch"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/safe"
	"gonum.org/v1/gonum/stat"
	"strings"
)

// This needs to be written so that it could work with newly created datasets,
// or datasets that are prexisting.
// where should I store datasets that are to be reused.

type TSCompare struct {
	ctx       context.Context
	bibleId   string
	mediaId   string
	testament request.Testament
	baseConn  db.DBAdapter
	conn      db.DBAdapter
}

func main() {
	ctx := context.Background()
	cliMisc := cli_misc.NewTSBucket(ctx)
	ntBooks := "3JN"
	var testament = request.Testament{NTBooks: []string{ntBooks}}
	tsDataPath := "cli_misc/find_timestamps/TestFilesetList.json"
	tsData := cliMisc.GetTSData(tsDataPath)
	var totalDiffs []float64
	for i, data := range tsData {
		if i > 200 {
			break
		}
		fmt.Println("Doing", data.MediaId)
		var status *log.Status
		var t TSCompare
		t.ctx = ctx
		t.bibleId = data.MediaId[:6]
		t.mediaId = data.MediaId
		t.testament = testament
		if t.HasTextAudio() {
			t.baseConn, status = t.LazyDataset(BBTS, ntBooks)
			if status != nil {
				fmt.Println(status)
				continue
			}
			t.conn, status = t.LazyDataset(Aeneas1, ntBooks)
			if status != nil {
				fmt.Println(status)
				continue
			}
			var diffs []float64
			diffs, status = t.CompareFileset()
			if status != nil {
				fmt.Println(status)
				continue
			}
			totalDiffs = append(totalDiffs, diffs...)
		} else {
			fmt.Println("Skip", data.MediaId)
		}
	}
	mean, stddev := stat.MeanStdDev(totalDiffs, nil)
	fmt.Println("Final", mean, stddev)
}

func (t *TSCompare) HasTextAudio() bool {
	client := fetch.NewAPIDBPClient(t.ctx, t.bibleId)
	info, status := client.BibleInfo()
	if status != nil {
		return false
	}
	var req request.Request
	req.TextData.BibleBrain.TextPlainEdit = true
	req.AudioData.BibleBrain.MP3_64 = true // try also 16 for better STT
	req.Testament.NT = true
	client.FindFilesets(&info, req.AudioData.BibleBrain, req.TextData.BibleBrain, req.Testament)
	return info.TextNTUSXFileset.Id != `` && info.TextNTPlainFileset.Id != `` && info.AudioNTFileset.Id != ``
}

// LazyDataset is a lazy dataset creation method. Create if it does not exist.
// The user name and the database name define it.
func (t *TSCompare) LazyDataset(yaml string, books string) (db.DBAdapter, *log.Status) {
	var conn db.DBAdapter
	ctx := context.Background()
	var req = strings.Replace(yaml, `{bibleId}`, t.bibleId, 3)
	req2 := strings.Replace(req, "{books}", books, 1)
	reqBytes := []byte(req2)
	decoder := decode_yaml.NewRequestDecoder(ctx)
	rq, status := decoder.Decode(reqBytes)
	if status != nil {
		return conn, status
	}
	if !db.DatabaseExists(rq.Username, rq.DatasetName) {
		var control = controller.NewController(ctx, reqBytes)
		filename, status := control.Process()
		if status != nil {
			return conn, status
		}
		fmt.Println("Created", filename)
	}
	return db.NewerDBAdapter(ctx, false, rq.Username, rq.DatasetName)
}

func (t *TSCompare) CompareFileset() ([]float64, *log.Status) {
	var totalDiffs []float64
	var status *log.Status
	for _, bookId := range db.RequestedBooks(t.testament) {
		lastChapter, _ := db.BookChapterMap[bookId]
		for chap := 1; chap <= lastChapter; chap++ {
			var diffs []float64
			diffs, status = t.CompareChapter(bookId, chap)
			if status != nil {
				return totalDiffs, status
			}
			mean, stddev := stat.MeanStdDev(diffs, nil)
			fmt.Println(t.mediaId, bookId, chap, mean, stddev)
			totalDiffs = append(totalDiffs, diffs...)
		}
	}
	mean, stddev := stat.MeanStdDev(totalDiffs, nil)
	fmt.Println("Total", t.mediaId, mean, stddev)
	return totalDiffs, status
}

func (t *TSCompare) CompareChapter(bookId string, chapter int) ([]float64, *log.Status) {
	var diffs []float64
	var baseTS, status = t.baseConn.SelectScriptTimestamps(bookId, chapter)
	if status != nil {
		return diffs, status
	}
	baseMap := make(map[int]db.Timestamp)
	for _, ts := range baseTS {
		baseMap[safe.SafeVerseNum(ts.VerseStr)] = ts
	}
	var compTS []db.Timestamp
	compTS, status = t.conn.SelectScriptTimestamps(bookId, chapter)
	if status != nil {
		return diffs, status
	}
	for _, cts := range compTS {
		bts, ok := baseMap[safe.SafeVerseNum(cts.VerseStr)]
		if !ok && bts.VerseStr != `0` {
			fmt.Println("Not found ", t.mediaId, bookId, chapter, bts.VerseStr)
			fmt.Println("baseTS", baseTS)
			fmt.Println("compTS", compTS)
		} else {
			diff := bts.BeginTS - cts.BeginTS
			fmt.Println("verse:", cts.VerseStr, "Base:", bts.BeginTS, "Compare:", cts.BeginTS, "diff:", diff)
			diffs = append(diffs, diff)
		}
	}
	return diffs, nil
}
