package align

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"os"
	"path/filepath"
	"testing"
)

func TestAlignWriter(t *testing.T) {
	ctx := context.Background()
	dataset := "N2ENGWEB"
	dbDir := filepath.Join(os.Getenv("GOPROJ"), "match")
	conn := db.NewDBAdapter(ctx, filepath.Join(dbDir, "N2ENGWEB.db"))
	asrConn := db.NewDBAdapter(ctx, filepath.Join(dbDir, "N2ENGWEB_audio.db"))
	calc := NewAlignSilence(ctx, conn, asrConn)
	audioDir := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), "ENGWEB", "ENGWEBN2DA-mp3-64")
	faLines, filenameMap, status := calc.Process(audioDir)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(len(faLines), len(filenameMap))
	writer := NewAlignWriter(ctx, conn)
	filename, status := writer.WriteReport(dataset, faLines, filenameMap)
	fmt.Println("Report Filename", filename)
	revisedName := filepath.Join(os.Getenv("GOPROJ"), "match", dataset+".html")
	_ = os.Rename(filename, revisedName)
	fmt.Println("Report Filename", revisedName)
}
