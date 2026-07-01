package tests

import (
	"testing"

	log "github.com/artificial-polyglot/arti/logger"
)

const runAnything = `is_new: yes
dataset_name: N2QAEBSP
username: GaryNTest
language_iso: qae
notify_ok: [ntfy/artificial-polyglot]
notify_err: [ntfy/artificial-polyglot,gary@shortsands.com]
text_data:
  file: /Users/gary/arti2/fcbh_data/Dawasamu N2QAEBSP (Gospels)/Text Files/SFM Text/*.SFM
audio_data:
  file: /Users/gary/arti2/fcbh_data/Dawasamu N2QAEBSP (Gospels)/N2QAEBSP Chapter VOX/*.mp3
timestamps:
  mms_fa_verse: yes
  mms_align: no
training:
  redo_training: yes
  mms_adapter:
    batch_mb: 4
    num_epochs: 1
    learning_rate: 1e-3
    warmup_pct: 12.0
    grad_norm_max: 0.4
`

func TestRunAnything(t *testing.T) {
	var yaml = runAnything
	log.SetOutput("stderr")
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
