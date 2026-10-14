package train

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	req "github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/ffmpeg"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
	"github.com/artificial-polyglot/arti/utility/stdio_exec"
)

const pctExcludePruneData = 5
const stdDevOverMeanPruneSilence = 1.5
const Wav2Vec2CTC = "wav2vec2ctc"   // belongs in request
const MMS300m = "facebook/mms-300m" // belongs in request
const MMS1b = "facebook/mms-1b"     // belongs in request

type Wav2Vec2Trainer struct {
	ctx       context.Context
	conn      db.DBAdapter
	langISO   string
	modelBase string
	args      req.Training2
}

func NewWav2Vec2Trainer(ctx context.Context, conn db.DBAdapter, langISO string, train req.Training2) Wav2Vec2Trainer {
	var t Wav2Vec2Trainer
	t.ctx = ctx
	t.conn = conn
	ident, status := t.conn.SelectIdent()
	fmt.Println("Status: ", status)
	fmt.Println("Ident: ", ident)
	scripts, status := t.conn.SelectScripts()
	fmt.Println("Status: ", status, "Len Scripts: ", len(scripts))
	t.langISO = langISO
	t.modelBase = train.ModelBase
	t.args = train
	return t
}

func (t *Wav2Vec2Trainer) HasModel() bool {
	client, status := s3_datastore.NewS3Client(t.ctx)
	if status != nil {
		log.Warn(t.ctx, status, "Failed to create S3 client checking for existing model")
		return false
	}
	bucket := os.Getenv("FCBH_MODELS_BUCKET")
	prefix := t.modelBase + "/" + t.langISO
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
func (t *Wav2Vec2Trainer) VerifyTrained(since time.Time) *log.Status {
	path := t.GetLocalTensorPath()
	info, err := os.Stat(path)
	if err != nil {
		return log.Error(t.ctx, 500, err, t.modelBase+" training did not produce a safetensor file", path)
	}
	if info.Size() <= 1000000 { // must be GT 1Meg, like mms.HasLocalAdapter
		return log.ErrorNoErr(t.ctx, 500, path+" file is too small to be a model", path, info.Size())
	}
	if info.ModTime().Before(since) {
		return log.ErrorNoErr(t.ctx, 500, path+" training did not write a new safetensor; file is older than this run",
			path, info.ModTime())
	}
	return nil
}

func (t *Wav2Vec2Trainer) Train(files []generic.InputFile) *log.Status {
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
	status := SilencePruner(t.ctx, t.conn, stdDevOverMeanPruneSilence)
	if status != nil {
		return status
	}
	status = dataPruner(t.conn, pctExcludePruneData)
	pythonPath := os.Getenv(`FCBH_MMS_ADAPTER_PYTHON`)
	pythonScript := filepath.Join(os.Getenv("GOPROJ"), "wav2vec2/train/trainer.py")
	status = stdio_exec.RunScriptWithLogging(t.ctx, pythonPath, pythonScript,
		t.langISO,
		t.conn.DatabasePath,
		`'`+tempDir+`'`,
		t.modelBase,
		strconv.Itoa(t.args.BatchMB),
		strconv.Itoa(t.args.NumEpochs),
		strconv.FormatFloat(t.args.LearningRate, 'e', -1, 64),
		strconv.FormatFloat(t.args.WarmupPct, 'f', -1, 64),
		strconv.FormatFloat(t.args.GradNormMax, 'f', -1, 64),
		strconv.FormatFloat(t.args.MinAudioSec, 'f', -1, 64),
		strconv.Itoa(t.args.LoggingSteps))
	// logging
	return status
}

func (t *Wav2Vec2Trainer) GetLocalDir() string {
	modelName := strings.ReplaceAll(t.modelBase, "/", "-")
	dir := filepath.Join(os.Getenv("FCBH_DATASET_DB"), modelName, t.langISO)
	return dir
}

func (t *Wav2Vec2Trainer) GetLocalTensorPath() string {
	dir := t.GetLocalDir()
	path := filepath.Join(dir, "model.safetensors")
	return path
}

func (t *Wav2Vec2Trainer) GetS3ObjectPrefix() string {
	modelName := strings.ReplaceAll(t.modelBase, "/", "-")
	dir := filepath.Join(modelName, t.langISO)
	return dir
}
