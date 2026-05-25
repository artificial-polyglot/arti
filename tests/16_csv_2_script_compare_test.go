package tests

import (
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"testing"
)

const LoadBaseScript = `is_new: yes
dataset_name: LoadBaseScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output_file: 16__load_base_script.json
text_data:
  file: /Users/gary/FCBH2024/tugutil/TUJNTMN2ST.xlsm
testament:
  nt_books: [MRK]
`

const CSV2ScriptCompare = `is_new: yes
dataset_name: CSV2ScriptCompare_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output_file: 16__csv_2_script_compare.html
#output_file: 16__csv_2_script_compare_output.json
text_data:
  file: /Users/gary/FCBH2024/tugutil/TUJNTMN2TT.csv
testament:
  nt_books: [MRK]
compare:
  base_dataset: LoadBaseScript_{bibleId}
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

func TestCSV2ScriptCompare(t *testing.T) {
	var tests []CtlTest
	tests = append(tests, CtlTest{BibleId: "TUJNTM", Expected: 788, TextNtId: "TUJNTMN2ST",
		TextType: request.TextScript, AudioNTId: "", Language: "tuj"})
	DirectTestUtility(LoadBaseScript, tests, t)
	tests = nil
	tests = append(tests, CtlTest{BibleId: "TUJNTM", Expected: 788, TextNtId: "TUJNTMN2TT", // bibleId TUJNTM
		TextType: request.TextCSV, AudioNTId: "", Language: "tuj"})
	DirectTestUtility(CSV2ScriptCompare, tests, t)
}
