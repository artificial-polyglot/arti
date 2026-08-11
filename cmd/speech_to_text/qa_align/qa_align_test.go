package qa_align

import (
	"context"
	"os"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func TestQAAlign(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	user := request.GetTestUser()
	conn, status := db.NewerDBAdapter(ctx, false, user, "PlainTextEditScript_ENGWEB")
	if status != nil {
		t.Error(status)
	}
	asr := NewQAAlign(ctx, conn, "eng", "", false)
	var files []input.InputFile
	var file input.InputFile
	file.BookId = "MRK"
	file.Chapter = 1
	file.MediaId = "ENGWEBN2DA"
	file.Directory = os.Getenv("FCBH_DATASET_FILES") + "/ENGWEB/ENGWEBN2DA-mp3-64/"
	file.Filename = "B02___01_Mark________ENGWEBN2DA.mp3"
	//file.MediaId = "ENGESVN1DA"
	//file.Directory = os.Getenv("FCBH_DATASET_FILES") + "/ENGESV/ENGESVN1DA/"
	//file.Filename = "B02___01_Mark________ENGESVN1DA.mp3"
	files = append(files, file)
	status = input.InsertAudioFiles(conn, files)
	if status != nil {
		t.Error(status)
	}
	status = asr.ProcessFiles()
	if status != nil {
		t.Error(status)
	}
}
