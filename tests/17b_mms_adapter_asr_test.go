package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

// This is not a test that is expected to run to completion.
// It exists so that one can debug the initial parts of training
// Monitor the process on $FCBH_DATASET_DB/dataset.log
// This template says nt_books: [PHM], but I don't think the training module has the ability

const mmsAdapterASR = `is_new: no
dataset_name: 17a_mms_adapter
username: Tests
language_iso: atg
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
testament:
  nt_books: [3JN] # 1JN, 2JN, 3JN, JUD all in test set
database:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/17a_mms_adapter.db
audio_data:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/*.mp3
speech_to_text:
  adapter_asr: y
compare:
  html_report: yes
  compare_settings: 
    lower_case: y
    remove_prompt_chars: y
    remove_punctuation: y
    double_quotes: 
      remove: y
    apostrophe: 
      remove: y
    hyphen:
      remove: y
    diacritical_marks:
      normalize_nfkd: y
`

func TestMMSAdapterASR(t *testing.T) {
	courier.IsCourierTest = true
	log.SetOutput("stderr")
	var yaml = mmsAdapterASR
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
