package align

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
)

func TestAlignWriter(t *testing.T) {
	ctx := context.Background()
	dataset := "N2ENGWEB"
	dbDir := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "Tests")
	conn := db.NewDBAdapter(ctx, filepath.Join(dbDir, "15a_mms_asr.db"))
	asrConn := db.NewDBAdapter(ctx, filepath.Join(dbDir, "15a_mms_asr_audio.db"))
	calc := NewAlignSilence(ctx, conn, asrConn)
	audioDir := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), "ENGWEB", "ENGWEBN2DA")
	faLines, filenameMap, status := calc.Process(audioDir)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(len(faLines), len(filenameMap))
	writer := NewAlignWriter(ctx, conn)
	filename, status := writer.WriteReport(dataset, faLines, filenameMap)
	fmt.Println("Report Filename", filename)
}
