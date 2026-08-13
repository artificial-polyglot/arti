package read

import (
	"context"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/input"
	"github.com/artificial-polyglot/arti/request"
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

func TestUSXParserAdHoc(t *testing.T) {
	ctx := context.Background()
	var file = generic.InputFile{Directory: "/Users/gary/Downloads", Filename: "044JHN.usx", BookId: "JHN"}
	var files []generic.InputFile
	files = append(files, file)
	var conn = db.NewDBAdapter(ctx, ":memory:")
	parser := NewUSXParser(conn)
	status := parser.ProcessFiles(files)
	if status != nil {
		t.Error(status)
	}
}
