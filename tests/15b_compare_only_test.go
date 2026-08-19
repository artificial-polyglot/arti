package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

// Test expects JMDYPM_audio.db and JMDYPM_text.db to exist.

const compareOnly = `is_new: no
dataset_name: 15b_compare_only_audio
bible_id: JMDYPM
username: Tests
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
database:
  aws_s3: s3://arti-output/Tests/15a_mms_asr/arti/00001/database/15a_mms_asr_audio.db
testament:
  nt: yes
compare:
  html_report: yes
  base_dataset: s3://arti-output/Tests/15a_mms_asr/arti/00001/database/15a_mms_asr.db
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

func TestTwoCompareDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 110})
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts WHERE script_begin_ts != 0.0", 105})
	DirectSqlTest(compareOnly, tests, t)
}

//func TestTwoCompareEnglishDirect(t *testing.T) {
//	var yaml = compareOnly
//	yaml = strings.Replace(yaml, "JMDYPM_audio", "AudioWhisperJson_ENGWEB_STT", 1)
//	yaml = strings.Replace(yaml, "JMDYPM", "ENGWEB", 1)
//	yaml = strings.Replace(yaml, "[MAT,MRK,LUK,JHN,ACT]", "[PHM]", 1)
//	yaml = strings.Replace(yaml, "JMDYPM_text", "AudioWhisperJson_ENGWEB", 1)
//	DirectSqlTest(yaml, []SqliteTest{}, t)
//}
