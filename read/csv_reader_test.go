package read

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	"os"
	"path/filepath"
	"testing"
)

func TestCSVReader(t *testing.T) {
	ctx := context.Background()
	conn, status := db.NewerDBAdapter(ctx, true, `GaryNTest`, `tugutil_test`)
	if status != nil {
		t.Fatal(status)
	}
	reader := NewCSVReader(conn)
	var files []input.InputFile
	var file input.InputFile
	file.BookId = `MRK`
	file.Testament = `NT`
	file.FileExt = `csv`
	file.MediaId = `TUJNTMN2TT` // is TT correct
	file.Filename = "transcribed.csv"
	file.MediaType = ``
	file.Directory = filepath.Join(os.Getenv(`FCBH_DATASET_DB`), `tugutil`)
	files = append(files, file)
	status = reader.ProcessFiles(files)
	if status != nil {
		t.Fatal(status)
	}
}
