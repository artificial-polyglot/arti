package timestamp

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	"github.com/artificial-polyglot/arti/read"
	"os"
	"testing"
)

// This test is not working...
func TestTSBucket(t *testing.T) {
	ctx := context.Background()
	conn := db.NewDBAdapter(ctx, ":memory:")
	ts, status := NewTSBucket(ctx, conn)
	if status != nil {
		t.Fatal(status)
	}
	key, status := ts.GetKey(ScriptTS, `ENGWEBN2DA`, `REV`, 22)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(key)
	object, status := ts.GetObject(TSBucketName, key)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(string(object))
	timestamps, status := ts.GetTimestamps(ScriptTS, `ENGWEBN2DA`, `REV`, 22)
	if status != nil {
		t.Fatal(status)
	}
	for _, time := range timestamps {
		fmt.Println(time)
	}
}

func TestTSBucket_GetTimestamps(t *testing.T) {
	ctx := context.Background()
	conn := db.NewDBAdapter(ctx, ":memory:")
	ts, status := NewTSBucket(ctx, conn)
	if status != nil {
		t.Fatal(status)
	}
	key, status := ts.GetKey(VerseAeneas, `ENGWEBN2DA`, `REV`, 22)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(key)
	object, status := ts.GetObject(TSBucketName, key)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(string(object))
	timestamps, status := ts.GetTimestamps(VerseAeneas, `ENGWEBN2DA`, `REV`, 22)
	if status != nil {
		t.Fatal(status)
	}
	for _, time := range timestamps {
		fmt.Println(time)
	}
}

func TestTSBucket_LoadTimestamps(t *testing.T) {
	ctx := context.Background()
	var database = "TestTSBucket_LoadTimestamps.db"
	db.DestroyDatabase(database)
	var conn = db.NewDBAdapter(ctx, database)
	ts, status := NewTSBucket(ctx, conn)
	if status != nil {
		t.Fatal(status)
	}
	var files []input.InputFile
	var file input.InputFile
	file.BookId = "MRK"
	file.Chapter = 1
	file.MediaId = "ENGWEBN2DA"
	file.Directory = os.Getenv("FCBH_DATASET_FILES") + "/ENGWEB/ENGWEBN_ET-usx/"
	file.Filename = "041MRK.usx"
	files = append(files, file)
	parser := read.NewUSXParser(conn)
	status = parser.ProcessFiles(files)
	if status != nil {
		t.Error(status)
	}
	status = ts.ProcessFiles(files)
	if status != nil {
		t.Error(status)
	}
}

//aws s3 ls s3://dbp-aeneas-staging/Latin_N2_organized/pass_qc/ENGWEBN2DA/cue_info_text

//aeneas_verse_timings

//aws s3 ls s3://dbp-aeneas-staging/Latin_N2_organized/pass_qc/ENGWEBN2DA/aeneas_verse_timings/
