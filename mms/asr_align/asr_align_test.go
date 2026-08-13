package asr_align

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func TestASRAlign_ProcessFiles(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	user := request.GetTestUser()
	conn, status := db.NewerDBAdapter(ctx, false, user, "N2MZJSIM")
	asr := NewASRAlign(ctx, conn, "mzj", "", false)
	asr.testing = true
	var files []generic.InputFile
	var file generic.InputFile
	file.BookId = "3JN"
	file.Chapter = 1
	file.MediaId = "N2MZJSIM"
	file.Directory = path.Join(os.Getenv("FCBH_DATASET_FILES"), "N2MZJSIM Manya (MZJ)", "N2MZJSIM Chapter VOX")
	file.Filename = "N2_MZJ_SIM_237_3JN_001_VOX.mp3"
	fmt.Println("audio file: ", file.FilePath())
	files = append(files, file)
	status = asr.ProcessFiles(files)
	if status != nil {
		t.Fatal(status)
	}
}
