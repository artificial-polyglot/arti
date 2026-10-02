package output

import (
	"context"
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/db"
)

func TestPrepareScripts(t *testing.T) {
	ctx := context.Background()
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	var out = NewOutput(ctx, conn, `TestScripts`, true, true)
	structs, meta := out.PrepareScripts()
	fmt.Println("Loaded Scripts", len(structs))
	filename, status := out.WriteCSV(structs, meta)
	if status != nil {
		t.Error(status)
	}
	fmt.Println("CoSV File", filename)
	filename, status = out.WriteJSON(structs, meta)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("JSON File", filename)
}

func TestPrepareWords(t *testing.T) {
	ctx := context.Background()
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	var out = NewOutput(ctx, conn, `TestWords`, true, true)
	structs, meta := out.PrepareWords()
	fmt.Println("Loaded Scripts", len(structs))
	filename, status := out.WriteCSV(structs, meta)
	if status != nil {
		t.Error(status)
	}
	fmt.Println("CSV File", filename)
	filename, status = out.WriteJSON(structs, meta)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("JSON File", filename)
}
