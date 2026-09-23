package proofing_rpt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
)

func TestTimestampWriter(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), "GaryNTest", "N1SKNSEC.db")
	conn := db.NewDBAdapter(ctx, dbPath)
	_ = conn.Database
	verses, status := SelectCharData(conn)
	if status != nil {
		t.Fatal(status)
	}
	output := TimestampWriters(ctx, verses)
	for _, out := range output {
		fmt.Println(out.Report, out.FilePath)
	}
}
