package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const mmsAdapter = `is_new: no
dataset_name: 17a_mms_train_adapter
username: Tests
language_iso: atx
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
testament:
  nt_books: [3JN] # All of 1JN,2JN,3JN,JUD are available
database:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/17a_mms_adapter.db
audio_data:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/*.mp3
training:
  redo_training: yes
  mms_adapter:
    batch_mb: 3
    num_epochs: 1
    learning_rate: 1e-3
    warmup_pct: 12.0
    grad_norm_max: 0.4
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

func TestMMSAdapter(t *testing.T) {
	courier.IsCourierTest = true
	var yaml = mmsAdapter
	DirectSqlTest(yaml, []SqliteTest{}, t)
}

/*
*
Test to prepare database
*/
const prepareDB = `is_new: yes
dataset_name: 17a_mms_adapter
language_iso: atg
username: Tests
output:
  directory: ~/Downloads
  csv: yes
  sqlite: yes
text_data:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/*.SFM
audio_data:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/*.mp3
detail:
  words: yes
timestamps:
  mms_align: yes
`

func PrepTestPrepareMMSAdapterDB(t *testing.T) {
	var yaml = prepareDB
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
