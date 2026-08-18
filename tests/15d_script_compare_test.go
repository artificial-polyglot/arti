package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const scriptCompare = `is_new: no
dataset_name: 15d_script_compare_audio
bible_id: IKHMLT
username: Tests
notify_err: [ntfy/arti2]
output:
  directory: ~/Download
  csv: yes
database:
  aws_s3: s3://arti-input/15d_script_compare_audio_test_data/15d_script_compare_audio.db
testament:
  nt: yes
compare:
  html_report: yes
  base_dataset: s3://arti-input/15d_script_compare_audio_test_data/15d_script_compare.db
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
      normalize_nfc: y
`

func TestScriptCompare(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 5112})
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts WHERE script_begin_ts != 0.0", 4995})
	DirectSqlTest(scriptCompare, []SqliteTest{}, t)
}
