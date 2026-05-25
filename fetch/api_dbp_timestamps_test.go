package fetch

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"testing"
)

func TestHavingTimestamps(t *testing.T) {
	ctx := context.Background()
	bibleId := `ENGWEB`
	database := bibleId + `_EDITTEXT.db`
	conn := db.NewDBAdapter(ctx, database)
	filesetId := `ENGWEBN2DA`
	api := NewAPIDBPTimestamps(conn, filesetId)
	timestampMap, status := api.HavingTimestamps()
	if status != nil {
		t.Error(status)
	}
	if len(timestampMap) != 344 {
		t.Error(`344 filesetIds are expected`, len(timestampMap))
	}
	_, ok := timestampMap[`ENGWEBN2DA`]
	if !ok {
		t.Error(`ENGWEBN2DA is expected in timestampMap`)
	}
	_, ok = timestampMap[`ENGWEBN1DA`]
	if ok {
		t.Error(`ENGWEBN2DA is not expected in timestampMap`)
	}
}

func TestTimestamps(t *testing.T) {
	ctx := context.Background()
	bibleId := `ENGWEB`
	database := bibleId + `_EDITTEXT.db`
	conn := db.NewDBAdapter(ctx, database)
	filesetId := `ENGWEBN2DA`
	api := NewAPIDBPTimestamps(conn, filesetId)
	timestamps, status := api.Timestamps(`MAT`, 5)
	if status != nil {
		t.Error(status)
	}
	if len(timestamps) != 49 {
		t.Error(`344 timestamp is expected`, len(timestamps))
	}
	last := timestamps[len(timestamps)-1]
	if last.VerseStart != `48` {
		t.Error(`Last verse is expected to be 48`, last.VerseStart)
	}
	if last.Timestamp != 390.98 {
		t.Error(`Last timestamp is expected to be 390.98`, last.Timestamp)
	}
	conn.Close()
}

func TestLoadTimestamps(t *testing.T) {
	ctx := context.Background()
	bibleId := `ENGWEB`
	database := bibleId + `_EDITTEXT`
	conn, status := db.NewerDBAdapter(ctx, false, `GaryNGriswold`, database)
	if status != nil {
		t.Fatal(status)
	}
	filesetId := `ENGWEBN2DA`
	testament := request.Testament{NTBooks: []string{`MAT`, `MRK`}}
	testament.BuildBookMaps()
	api := NewAPIDBPTimestamps(conn, filesetId)
	ok, status := api.LoadTimestamps(testament)
	if status != nil {
		t.Error(status)
	}
	if !ok {
		t.Error(`ok should be true`)
	}
}
