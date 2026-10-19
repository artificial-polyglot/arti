package wav2vec2

import (
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/wav2vec2/train"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	ctx := database.Ctx
	var output []db.Output
	files, status := db.SelectAudioFiles(database)
	if status != nil {
		return output, status
	}
	trainer := train.NewWav2Vec2Trainer(ctx, database, req.LanguageISO, req.Training2)
	err := trainer.Train(files)
	if err != nil {
		return output, log.Error(ctx, 500, err, "Error in Trainer2")
	}
	return output, nil
}

/*


func TestNewWav2Vec2Trainer(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(os.Getenv("HOME"), "Downloads", "arti-output_GaryNTest_N2TTS_arti_00011_database_N2TTS.db")
	database := db.NewDBAdapter(ctx, dbPath)
	files, status := input.FileInput(ctx, "/Users/gary/arti2/isanbible/*.mp3")
	if status != nil {
		t.Fatal(status)
	}
	params := req.Training2{ModelBase: "wav2vec2ctc", BatchMB: 4, NumEpochs: 1, LearningRate: 1e-3, WarmupPct: 12, GradNormMax: 0.4, LoggingSteps: 2}
	trainer := NewWav2Vec2Trainer(ctx, database, "tts", params)
	err := trainer.Train(files)
	if err != nil {
		t.Fatal(err)
	}
}
*/
