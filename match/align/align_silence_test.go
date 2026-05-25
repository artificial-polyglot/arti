package align

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAlignSilence(t *testing.T) {
	ctx := context.Background()
	dbDir := filepath.Join(os.Getenv("GOPROJ"), "match")
	conn := db.NewDBAdapter(ctx, filepath.Join(dbDir, "N2ENGWEB.db"))
	asrConn := db.NewDBAdapter(ctx, filepath.Join(dbDir, "N2ENGWEB_audio.db"))
	calc := NewAlignSilence(ctx, conn, asrConn)
	audioDir := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), "ENGWEB", "ENGWEBN2DA-mp3-64")
	faLines, filenameMap, status := calc.Process(audioDir)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println(len(filenameMap))
	calc.countErrors(faLines)
	fmt.Println(len(faLines))
}
