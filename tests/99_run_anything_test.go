package tests

import (
	"testing"

	log "github.com/artificial-polyglot/arti/logger"
)

const runAnything = `is_new: true 
dataset_name: N2TTS
username: GaryNTest
bible_id: ""
language_iso: tts 
priority: 3
notify_ok:
    - ntfy/arti2
notify_err:
    - ntfy/arti2
testament:
    nt: true
    ot: true
#database:
#    aws_s3: s3://arti-output/GaryNTest/N1SKNSEC/arti/00003/database/N1SKNSEC.db
audio_data:
    aws_s3: s3://arti-input/N2TTS_ISan/*.mp3
text_data:
    file: /Users/gary/arti2/laobible/LAO_BIBLE.json
#    aws_s3: s3://arti-input/N2TTS_ISan/*.json
timestamps:
    mms_align: true
training:
    redo_training: true 
    mms_adapter:
        batch_mb: 4
        num_epochs: 16
        learning_rate: 0.001
        warmup_pct: 12
        grad_norm_max: 0.4
audio_proof:
    html_report: true
`

func TestRunAnything(t *testing.T) {
	var yaml = runAnything
	log.SetOutput("stderr")
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
