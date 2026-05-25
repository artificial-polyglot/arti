package read

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"testing"
)

func TestDBPEditTextReader(t *testing.T) {
	var req request.Request
	req.BibleId = `ENGWEB`
	req.Testament = request.Testament{OTBooks: []string{`GEN`, `EXO`}, NTBooks: []string{`MAT`, `MRK`}}
	req.Testament.BuildBookMaps()
	var database = req.BibleId + `_EDITTEXT.db`
	db.DestroyDatabase(database)
	ctx := context.Background()
	var db1 = db.NewDBAdapter(ctx, database)
	reader := NewDBPTextEditReader(db1, req)
	status := reader.Process()
	if status != nil {
		t.Error(status)
	}
	count, status := db1.CountScriptRows()
	if status != nil {
		t.Error(status)
	}
	if count != 4629 {
		t.Error(`Expected count to be 4629`, count)
	}
}
