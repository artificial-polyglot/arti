package read

import (
	"context"
	"testing"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
)

func TestUSFMParser(t *testing.T) {
	ctx := context.Background()
	var directory = "test_data"
	var files []input.InputFile
	file1 := input.InputFile{BookId: "", Directory: directory, Filename: "43LUKCFM.SFM", MediaType: "text/plain"}
	//file2 := input.InputFile{BookId: "", Directory: directory, Filename: "43LUKDWK.SFM", MediaType: "text/plain"}
	files = append(files, file1)
	var database = directory + "/usfm_test.db"
	db.DestroyDatabase(database)
	var conn = db.NewDBAdapter(ctx, database)
	parser := NewUSFMParser(conn)
	status := parser.ProcessFiles(files)
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
