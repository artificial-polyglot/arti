package tests

import (
	"testing"

	log "github.com/artificial-polyglot/arti/logger"
)

// This is not a test that is expected to run to completion.
// It exists so that one can debug the initial parts of training
// Monitor the process on $FCBH_DATASET_DB/dataset.log
// This template says nt_books: [PHM], but I don't think the training module has the ability

const qaAlignRpt = `is_new: no
dataset_name: 18a_qa_align_rpt_test.go
username: GaryNTest
language_iso: qae
notify_ok: [ntfy/arti2]
notify_err: [ntfy/arti2]
#testament:
#  nt_books: [MRK]
database:
  aws_s3: s3://arti-output/GaryNTest/N2QAEBSP/arti/00001/database/N2QAEBSP.db
audio_data:
  aws_s3: s3://arti-input/N2QAEBSP/N2QAEBSP Chapter VOX/*.mp3
audio_proof:
  html_report: yes
`

func TestQAAlignTest(t *testing.T) {
	log.SetOutput("stderr")
	var yaml = qaAlignRpt
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
