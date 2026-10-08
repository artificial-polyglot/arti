package train

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	req "github.com/artificial-polyglot/arti/request"
)

func TestNewWav2Vec2Trainer(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(os.Getenv("HOME"), "Downloads", "arti-output_GaryNTest_N2TTS_arti_00011_database_N2TTS.db")
	database := db.NewDBAdapter(ctx, dbPath)
	files, status := input.FileInput(ctx, "/Users/gary/arti2/isanbible/*.mp3")
	if status != nil {
		t.Fatal(status)
	}
	params := req.Wav2Vec2{BatchMB: 4, NumEpochs: 1, LearningRate: 1e-3, WarmupPct: 12, GradNormMax: 0.4}
	trainer := NewWav2Vec2Trainer(ctx, database, "tts", params)
	err := trainer.Train(files)
	if err != nil {
		t.Fatal(err)
	}
}
