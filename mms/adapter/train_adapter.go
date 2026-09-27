package adapter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	req "github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/ffmpeg"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
	"github.com/artificial-polyglot/arti/utility/stdio_exec"
)

type TrainAdapter struct {
	ctx     context.Context
	conn    db.DBAdapter
	langISO string
	args    req.MMSAdapter
}

func NewTrainAdapter(ctx context.Context, conn db.DBAdapter, langISO string, train req.MMSAdapter) TrainAdapter {
	var t TrainAdapter
	t.ctx = ctx
	t.conn = conn
	ident, status := t.conn.SelectIdent()
	fmt.Println("Status: ", status)
	fmt.Println("Ident: ", ident)
	scripts, status := t.conn.SelectScripts()
	fmt.Println("Status: ", status, "Len Scripts: ", len(scripts))
	t.langISO = langISO
	t.args = train
	return t
}

func (t *TrainAdapter) HasModel() bool {
	client, status := s3_datastore.NewS3Client(t.ctx)
	if status != nil {
		log.Warn(t.ctx, status, "Failed to create S3 client checking for existing model")
		return false
	}
	bucket := os.Getenv("FCBH_MODELS_BUCKET")
	prefix := "mms_adapters/" + t.langISO
	has, status := client.HasModel(bucket, prefix)
	if status != nil {
		log.Warn(t.ctx, status, "Failed to check R2 for existing model", prefix)
		return false
	}
	return has
}

// VerifyTrained checks that Train really produced an adapter: the adapter file must exist, be
// larger than 1 MB, and have been written at or after since (the time Train was started).
// Train returns nil without training when there are no audio files, and a stale adapter
// left in the local directory must never be published as a newly trained model.
func (t *TrainAdapter) VerifyTrained(since time.Time) *log.Status {
	dir := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "mms_adapters", t.langISO)
	path := filepath.Join(dir, "adapter."+t.langISO+".safetensors")
	info, err := os.Stat(path)
	if err != nil {
		return log.Error(t.ctx, 500, err, "MMS adapter training did not produce an adapter file", path)
	}
	if info.Size() <= 1000000 { // must be GT 1Meg, like mms.HasLocalAdapter
		return log.ErrorNoErr(t.ctx, 500, "MMS adapter file is too small to be a trained adapter", path, info.Size())
	}
	if info.ModTime().Before(since) {
		return log.ErrorNoErr(t.ctx, 500, "MMS adapter training did not write a new adapter; file is older than this run",
			path, info.ModTime())
	}
	return nil
}

func (t *TrainAdapter) Train(files []generic.InputFile) *log.Status {
	if len(files) == 0 {
		return nil
	}
	tempDir := files[0].Directory
	for _, file := range files {
		_, status := ffmpeg.ConvertMp3ToWav(t.ctx, tempDir, file.FilePath())
		if status != nil {
			return status
		}
	}
	status := SilencePruner(t.ctx, 400, t.conn)
	if status != nil {
		return status
	}
	pythonPath := os.Getenv(`FCBH_MMS_ADAPTER_PYTHON`)
	pythonScript := filepath.Join(os.Getenv("GOPROJ"), "mms/adapter/trainer.py")
	status = stdio_exec.RunScriptWithLogging(t.ctx, pythonPath, pythonScript,
		t.langISO,
		t.conn.DatabasePath,
		`'`+tempDir+`'`,
		strconv.Itoa(t.args.BatchMB),
		strconv.Itoa(t.args.NumEpochs),
		strconv.FormatFloat(t.args.LearningRate, 'e', -1, 64),
		strconv.FormatFloat(t.args.WarmupPct, 'f', -1, 64),
		strconv.FormatFloat(t.args.GradNormMax, 'f', -1, 64))
	return status
}
