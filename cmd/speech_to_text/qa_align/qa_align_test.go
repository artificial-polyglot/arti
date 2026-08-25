package qa_align

import (
	"context"
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func TestQAAlign(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	database, status := input.AWSS3Input(ctx, "s3://arti-output/GaryNTest/N1SKNSEC/arti/00010/database/*.db")
	if status != nil {
		t.Fatal(status)
	}
	conn := db.NewDBAdapter(ctx, database[0].FilePath())
	fmt.Println("Database Path", database[0].FilePath())
	var testament request.Testament
	testament.BuildBookMaps()
	testament.NT = true
	testament.OT = true
	asr := NewQAAlign(ctx, conn, "skn", "", true, testament)
	files, status := input.AWSS3Input(ctx, "s3://arti-input/Kolibugan N1SKNSEC/N1SKNSEC Chapter mp3/*MRK_001_VOX.mp3")
	if status != nil {
		t.Fatal(status)
	}
	files[0].BookId = "MRK"
	files[0].Chapter = 1
	status = db.InsertAudioFiles(conn, files)
	if status != nil {
		t.Error(status)
	}
	status = asr.ProcessFiles()
	if status != nil {
		t.Error(status)
	}
}
