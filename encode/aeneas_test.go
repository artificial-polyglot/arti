package encode

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/input"
	"testing"
)

func TestAeneasLines(t *testing.T) {
	var ctx = context.Background()
	var bibleId = `ENGWEB`
	var filesetId = `ENGWEBN2DA`
	var language = `eng`
	var testament = request.Testament{NTBooks: []string{`MRK`}}
	testament.BuildBookMaps()
	var detail = request.Detail{Lines: true}
	files, status := input.DBPDirectory(ctx, bibleId, `audio`, ``, filesetId, testament)
	if status != nil {
		t.Error(status)
	}
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	aeneas := NewAeneas(ctx, conn, bibleId, language, detail)
	status = aeneas.ProcessFiles(files)
	if status != nil {
		t.Error(status)
	}
}

func TestAeneasWords(t *testing.T) {
	var ctx = context.Background()
	var bibleId = `ENGWEB`
	var filesetId = `ENGWEBN2DA`
	var language = `eng`
	var testament = request.Testament{NT: true}
	testament.BuildBookMaps()
	var detail = request.Detail{Words: true}
	files, status := input.DBPDirectory(ctx, bibleId, `audio`, ``, filesetId, testament)
	if status != nil {
		t.Error(status)
	}
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	aeneas := NewAeneas(ctx, conn, bibleId, language, detail)
	status = aeneas.ProcessFiles(files)
	if status != nil {
		t.Error(status)
	}
}
