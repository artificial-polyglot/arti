package timestamp

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"io"
	"os"
	"strconv"
	"strings"
)

/*
This class downloads timestamps and other data from Sandeep's timestamp bucket.
It provides methods to access Sandeep's bucket of timestamp data
*/

const (
	TSBucketName = `dbp-aeneas-staging`
	LatinN1      = `Latin_N1_organized/pass_qc/`
	LatinN2      = `Latin_N2_organized/pass_qc/`
	Script       = `core_script/`
	ScriptTS     = `cue_info_text/`
	LineAeneas   = `aeneas_line_timings/`
	VerseAeneas  = `aeneas_verse_timings/`
)

type TSBucket struct {
	ctx    context.Context
	conn   db.DBAdapter
	client *s3.Client
}

func NewTSBucket(ctx context.Context, conn db.DBAdapter) (TSBucket, *log.Status) {
	var t TSBucket
	t.ctx = ctx
	t.conn = conn
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		status := log.Error(ctx, 400, err, "Failed to load AWS S3 Config.")
		return t, status
	}
	t.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "us-west-2"
	})
	return t, nil
}

func (t *TSBucket) ListObjects(bucket, prefix string) ([]string, *log.Status) {
	var results []string
	list, err := t.client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		status := log.Error(t.ctx, 500, err, "Error listing S3 bucket objects.")
		return results, status
	}
	for _, obj := range list.Contents {
		key := aws.ToString(obj.Key)
		results = append(results, key)
	}
	return results, nil
}

func (t *TSBucket) ListPrefix(bucket, prefix string) ([]string, *log.Status) {
	var results []string
	list, err := t.client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(`/`),
	})
	if err != nil {
		status := log.Error(t.ctx, 500, err, "Failed to list S3 bucket objects.")
		return results, status
	}
	for _, obj := range list.CommonPrefixes {
		pref := aws.ToString(obj.Prefix)
		results = append(results, pref)
	}
	return results, nil
}

func (t *TSBucket) ProcessFiles(files []input.InputFile) *log.Status {
	for _, file := range files {
		//var scripts []db.Script
		scripts, status := t.conn.SelectScriptsByChapter(file.BookId, file.Chapter)
		if status != nil {
			return status
		}
		var verseMap = make(map[string]int)
		for _, scr := range scripts {
			verseMap[scr.VerseStr] = scr.ScriptId
		}
		var timestamps []db.Audio
		timestamps, status = t.GetTimestamps(VerseAeneas, file.MediaId, file.BookId, file.Chapter)
		if status != nil {
			return status
		}
		for i := range timestamps {
			scriptId, ok := verseMap[timestamps[i].VerseStr]
			if !ok {
				log.Warn(t.ctx, "Could not match verse "+timestamps[i].VerseStr)
			}
			timestamps[i].ScriptId = int64(scriptId)
		}
		status = t.conn.UpdateScriptFATimestamps(timestamps)
		if status != nil {
			return status
		}
	}
	return nil
}

func (t *TSBucket) GetTimestamps(tsType string, mediaId string, bookId string, chapterNum int) ([]db.Audio, *log.Status) {
	var results []db.Audio
	key, status := t.GetKey(tsType, mediaId, bookId, chapterNum)
	if status != nil {
		return results, status
	}
	object, status := t.GetObject(TSBucketName, key)
	if status != nil {
		return results, status
	}
	for i, row := range strings.Split(string(object), "\n") {
		if i == 0 { // Add an entry for chapter heading
			var ts db.Audio
			ts.BookId = bookId
			ts.ChapterNum = chapterNum
			ts.VerseStr = "0"
			ts.BeginTS = 0.0
			parts := strings.Split(row, "\t")
			ts.EndTS, _ = strconv.ParseFloat(parts[0], 64)
			results = append(results, ts)
		}
		parts := strings.Split(row, "\t")
		if len(parts) >= 3 {
			var ts db.Audio
			ts.BookId = bookId
			ts.ChapterNum = chapterNum
			ts.VerseStr = strings.TrimLeft(parts[2], `0`)
			ts.BeginTS, _ = strconv.ParseFloat(parts[0], 64)
			ts.EndTS, _ = strconv.ParseFloat(parts[1], 64)
			results = append(results, ts)
		}
	}
	return results, nil
}

func (t *TSBucket) GetObject(bucket string, key string) ([]byte, *log.Status) {
	response, err := t.client.GetObject(t.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return []byte{}, log.Error(t.ctx, 400, err, "Error getting S3 bucket object.")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return body, log.Error(t.ctx, 500, err, "Error reading S3 object body.")
	}
	return body, nil
}

func (t *TSBucket) DownloadObject(bucket string, key string, path string) *log.Status {
	content, status := t.GetObject(bucket, key)
	if status != nil {
		return status
	}
	err := os.WriteFile(path, content, 0644)
	if err != nil {
		return log.Error(t.ctx, 500, err, "DownloadObject failed.")
	}
	return nil
}

func (t *TSBucket) GetKey(tsType string, mediaId string, bookId string, chapterNum int) (string, *log.Status) {
	var result []string
	if mediaId[7] == '1' {
		result = append(result, LatinN1)
	} else {
		result = append(result, LatinN2)
	}
	result = append(result, mediaId)
	result = append(result, `/`)
	switch tsType {
	case LineAeneas:
		result = append(result, LineAeneas)
		result = append(result, t.GetAeneasKey(bookId, chapterNum))
	case VerseAeneas:
		result = append(result, VerseAeneas)
		result = append(result, t.GetAeneasKey(bookId, chapterNum))
	case ScriptTS:
		result = append(result, ScriptTS)
		result = append(result, t.GetScriptTSKey(mediaId, bookId, chapterNum))
	case Script:
		result = append(result, Script)
		prefix := strings.Join(result, "")
		//var list []string
		list, status := t.ListObjects(TSBucketName, prefix)
		if status != nil {
			return "", status
		}
		if len(list) != 1 {
			panic(`There should be 1 script file, but there are ` + strconv.Itoa(len(list)))
		}
		result = []string{list[0]}
	}
	return strings.Join(result, ""), nil
}

func (t *TSBucket) GetAeneasKey(bookId string, chapterNum int) string {
	var result []string
	result = append(result, `C01`)
	seq := db.BookSeqMap[bookId] - 40
	seqStr := strconv.Itoa(seq)
	if len(seqStr) < 2 {
		seqStr = "0" + seqStr
	}
	result = append(result, seqStr)
	result = append(result, bookId)
	chapStr := strconv.Itoa(chapterNum)
	if len(chapStr) < 2 {
		chapStr = "0" + chapStr
	}
	result = append(result, chapStr)
	result = append(result, `timing.txt`)
	return strings.Join(result, `-`)
}

func (t *TSBucket) GetScriptTSKey(mediaId string, bookId string, chapterNum int) string {
	//N2_APF_CMU_001_MAT_01_cue_info.txt
	var result []string
	result = append(result, mediaId[6:8])
	result = append(result, mediaId[0:3])
	result = append(result, mediaId[3:6])
	sequence := t.chapterSeq(bookId, chapterNum)
	seqStr := strconv.Itoa(sequence)
	if len(seqStr) < 2 {
		seqStr = "00" + seqStr
	} else if len(seqStr) < 3 {
		seqStr = "0" + seqStr
	}
	result = append(result, seqStr)
	result = append(result, bookId)
	chapStr := strconv.Itoa(chapterNum)
	if len(chapStr) < 2 {
		chapStr = "0" + chapStr
	}
	result = append(result, chapStr)
	//result = append(result, `VOX.clt_cue_info.txt`)
	result = append(result, `cue_info.txt`)
	return strings.Join(result, `_`)
}

func (t *TSBucket) chapterSeq(bookId string, chapterNum int) int {
	var seq = 0
	for _, book := range db.BookNT {
		if book == bookId {
			seq += chapterNum
			break
		} else {
			chaps := db.BookChapterMap[book]
			seq += chaps
		}
	}
	return seq
}
