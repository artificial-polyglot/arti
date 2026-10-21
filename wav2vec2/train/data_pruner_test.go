package train

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func TestDataPruner(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	dbPath := filepath.Join(os.Getenv("HOME"), "Downloads", "arti-output_GaryNTest_N2TTS_arti_00011_database_N2TTS.db")
	conn := db.NewDBAdapter(ctx, dbPath)
	status := WordDataPruner(conn, 5)
	if status != nil {
		t.Fatal(status)
	}
}
