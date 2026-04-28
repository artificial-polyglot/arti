package read

import (
	"context"
	"testing"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/decode_yaml/request"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
)

func TestDBPTextReader1(t *testing.T) {
	ctx := context.Background()
	bibleId := `ENGWEB`
	fsType := request.TextPlainEdit
	otFileset := `ENGWEBO_ET`
	ntFileset := `ENGWEBN_ET`
	testament := request.Testament{NTBooks: []string{`MAT`, `MRK`}, OTBooks: []string{`JOB`, `PSA`, `PRO`, `SNG`}}
	files, status := input.DBPDirectory(ctx, bibleId, fsType, otFileset, ntFileset) //, testament)
	if status != nil {
		t.Error(status)
	}
	var database = bibleId + `_DBPTEXT.db`
	db.DestroyDatabase(database)
	var db1 = db.NewDBAdapter(context.Background(), database)
	var req request.Request
	req.Testament = testament
	req.Testament.BuildBookMaps()
	textAdapter := NewDBPTextReader(db1, req.Testament)
	textAdapter.ProcessFiles(files)
	count, _ := db1.CountScriptRows()
	if count != 6312 {
		t.Error(`Script row count should be 1`, count)
	}
	db1.Close()
}
