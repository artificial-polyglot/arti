package proofing_rpt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
)

func TestAlignWriter(t *testing.T) {
	ctx := context.Background()
	dataset := "N1SKNSEC"
	//dbDir := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "GaryNTest")

	database, status := input.AWSS3Input(ctx, "s3://arti-output/GaryNTest/N1SKNSEC/arti/00002/database/*.db")
	if status != nil {
		t.Fatal(status)
	}
	conn := db.NewDBAdapter(ctx, database[0].FilePath())
	asrConn := db.NewDBAdapter(ctx, database[1].FilePath())
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
