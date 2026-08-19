package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	"github.com/artificial-polyglot/arti/request"
)

const mMSASRCompare = `is_new: yes
dataset_name: 15a_mms_asr
bible_id: ENGWEB
username: Tests
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
text_data:
  bible_brain:
    text_usx_edit: yes
audio_data:
  aws_s3: s3://arti-input/15a_mms_asr_compare_test_data/*.mp3
timestamps:
  mms_align: yes
  bible_brain: no # bible_brain does not return filenames
testament:
  nt_books: ['1JN']
speech_to_text:
  mms_asr: yes
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
      normalize_nfd: y
`

func TestMMSASRCompare(t *testing.T) {
	courier.IsCourierTest = true
	var tests []CtlTest
	tests = append(tests, CtlTest{BibleId: "ENGWEB", Expected: 18, TextNtId: "ENGWEBN_ET-usx",
		TextType: request.TextUSXEdit, AudioNTId: "ENGWEBN2DA", Language: "eng"})
	//tests = append(tests, CtlTest{BibleId: "APFCMU", Expected: 16, TextNtId: "APFCMUN_ET-usx",
	//	AudioNTId: `APFCMUN1DA`, TextType: request.TextUSXEdit, Language: "apf"})
	DirectTestUtility(mMSASRCompare, tests, t)
}
