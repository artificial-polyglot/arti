package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const vesselTrainTest = `is_new: yes
dataset_name: 16f_vessel_test
username: Tests
language_iso: eng
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
testament:
  nt_books: [TIT]
text_data:
  aws_s3: s3://arti-input/16e_vessel_asr_test_data/16e_vessel_test.xlsx
audio_data:
  aws_s3: s3://arti-input/16e_vessel_asr_test_data/*_VOX.wav
timestamps:
  mms_align: y
speech_to_text:
  mms_asr: y
compare:
  html_report: yes
  gordon_filter: 4
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
      normalize_nfd: y
`

func TestVesselTrain(t *testing.T) {
	courier.IsCourierTest = true
	var yaml = vesselTrainTest
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
