package read

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/input"
	"testing"
)

func TestUSXParser(t *testing.T) {
	ctx := context.Background()
	var bibleId = `ENGWEB`
	fsType := request.TextUSXEdit
	otFileset := `ENGWEBO_ET-usx`
	ntFileset := `ENGWEBN_ET-usx`
	testament := request.Testament{NTBooks: []string{`MAT`, `MRK`}, OTBooks: []string{`JOB`, `PSA`, `PRO`, `SNG`}}
	testament.BuildBookMaps()
	files, status := input.DBPDirectory(ctx, bibleId, fsType, otFileset, ntFileset) //, testament)
	if status != nil {
		t.Error(status)
	}
	var database = bibleId + `_USXEDIT.db`
	db.DestroyDatabase(database)
	var conn = db.NewDBAdapter(ctx, database)
	parser := NewUSXParser(conn)
	status = parser.ProcessFiles(files)
	if status != nil {
		t.Fatal(status)
	}
	count, stat2 := conn.CountScriptRows()
	if stat2 != nil {
		t.Error(stat2)
	}
	if count != 11755 {
		t.Error(`Expected 11755, but got`, count)
	}
	conn.Close()
}
