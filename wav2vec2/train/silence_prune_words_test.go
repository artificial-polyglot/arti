package train

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func TestSilencePruner(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	dbPath := filepath.Join(os.Getenv("HOME"), "Downloads", "arti-output_GaryNTest_N2TTS_arti_00011_database_N2TTS.db")
	conn := db.NewDBAdapter(ctx, dbPath)
	status := WordSilencePruner(ctx, conn, 1.5)
	if status != nil {
		t.Fatal(status)
	}
	rows, err := conn.DB.Query("SELECT word_id FROM pruned_silence")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var count int
	var wordId int64
	for rows.Next() {
		rows.Scan(&wordId)
		fmt.Println(wordId)
		count++
	}
	err = rows.Err()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("count", count)
}
