package encode

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/input"
	"github.com/artificial-polyglot/arti/read"
	"testing"
)

func TestFastText(t *testing.T) {
	var ctx = context.Background()
	db.DestroyDatabase(`ENGWEB_DBPTEXT.db`)
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	bibleId := `ENGWEB`
	testament := request.Testament{NT: true}
	files, status := input.DBPDirectory(ctx, bibleId, `text_plain`, `ENGWEBO_ET`,
		`ENGWEBN_ET`, testament)
	reader := read.NewDBPTextReader(conn, testament)
	status = reader.ProcessFiles(files)
	if status != nil {
		t.Error(status)
	}
	words := read.NewWordParser(conn)
	status = words.Parse()
	if status != nil {
		t.Error(status)
	}
	var fast = NewFastText(ctx, conn)
	status = fast.Process()
	if status != nil {
		t.Error(status)
	}
}
