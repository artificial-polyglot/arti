package cli_misc

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"io"
	"os"
	"strconv"
	"strings"
)

/*
This file has been copied to the timestamp directory
This file was written to support testing.
It provides methods to access Sandeep's bucket of timestamp data
If moved to production, there is significant error handling to be done.
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
	client *s3.Client
}

type TSData struct {
	MediaType    string `json:"media_type"`
	MediaId      string `json:"media_id"`
	PlainText    string `json:"plain_text"`
	ScriptPath   string `json:"script_path"`
	ScriptTSPath string `json:"script_ts_path"`
	LineTSPath   string `json:"line_ts_path"`
	VerseTSPath  string `json:"verse_ts_path"`
	Count        int    `json:"count"`
}

func NewTSBucket(ctx context.Context) TSBucket {
	var t TSBucket
	t.ctx = ctx
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	t.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "us-west-2"
	})
	return t
}

func (t *TSBucket) ListObjects(bucket, prefix string) []string {
	var results []string
	list, err := t.client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		panic(err)
	}
	for _, obj := range list.Contents {
		key := aws.ToString(obj.Key)
		results = append(results, key)
	}
	return results
}

func (t *TSBucket) ListPrefix(bucket, prefix string) []string {
	var results []string
	list, err := t.client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(`/`),
	})
	if err != nil {
		panic(err)
	}
	for _, obj := range list.CommonPrefixes {
		pref := aws.ToString(obj.Prefix)
		results = append(results, pref)
	}
	return results
}

func (t *TSBucket) GetTimestamps(tsType string, mediaId string, bookId string, chapterNum int) []db.Timestamp {
	var results []db.Timestamp
	key := t.GetKey(tsType, mediaId, bookId, chapterNum)
	object := t.GetObject(TSBucketName, key)
	for _, row := range strings.Split(string(object), "\n") {
		var ts db.Timestamp
		parts := strings.Split(row, "\t")
		if len(parts) >= 3 {
			ts.BeginTS, _ = strconv.ParseFloat(parts[0], 64)
			ts.EndTS, _ = strconv.ParseFloat(parts[1], 64)
			ts.VerseStr = strings.TrimLeft(parts[2], `0`)
			results = append(results, ts)
		}
	}
	return results
}

func (t *TSBucket) GetObject(bucket string, key string) []byte {
	response, err := t.client.GetObject(t.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Warn(t.ctx, err)
		return []byte{}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func (t *TSBucket) DownloadObject(bucket string, key string, path string) {
	content := t.GetObject(bucket, key)
	err := os.WriteFile(path, content, 0644)
	if err != nil {
		panic(err)
	}
}

func (t *TSBucket) GetTSData(filePath string) []TSData {
	//content, err := os.ReadFile("../cli_misc/find_timestamps/TestFilesetList.json")
	content, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	var tsData []TSData
	err = json.Unmarshal(content, &tsData)
	if err != nil {
		panic(err)
	}
	return tsData
}

func (t *TSBucket) GetKey(tsType string, mediaId string, bookId string, chapterNum int) string {
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
		list := t.ListObjects(TSBucketName, prefix)
		if len(list) != 1 {
			panic(`There should be 1 script file, but there are ` + strconv.Itoa(len(list)))
		}
		result = []string{list[0]}
	}
	return strings.Join(result, "")
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
