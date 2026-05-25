package tests

import (
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"testing"
)

const TextUSXEdit07 = `is_new: yes
dataset_name: 07_USXTextEditScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output:
  json: yes
text_data:
  bible_brain:
    text_usx_edit: yes
`

const TextScript07 = `is_new: yes
dataset_name: 07_ScriptTextScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output:
  csv: yes
text_data:
  file: /Users/gary/FCBH2024/download/ATIWBT/ATIWBTN2ST.xlsx
`

const TextScript07BGGWFW = `is_new: yes
dataset_name: 07_ScriptTextScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output:
  csv: yes
text_data:
  file: /Users/gary/FCBH2024/download/BGGWFW/BGGWFWN2ST.xlsx
`

const CompareUsXTextEdit2Script07 = `is_new: no
dataset_name: 07_ScriptTextScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
compare:
  html_report: yes
  base_dataset: 07_USXTextEditScript_{bibleId}
  compare_settings: # Mark yes, all settings that apply
    lower_case: n
    remove_prompt_chars: y
    remove_punctuation: y
    double_quotes: 
      remove: y
      normalize:
    apostrophe: 
      remove: y
      normalize:
    hyphen:
      remove: y
      normalize:
    diacritical_marks: 
      remove:
      normalize_nfc: 
      normalize_nfd: y
      normalize_nfkc:
      normalize_nfkd:
`

func TestCompareUsXTextEdit2ScriptAPI(t *testing.T) {
	var cases []APITest
	cases = append(cases, APITest{BibleId: `ATIWBT`, Expected: 2})
	APITestUtility(CompareUsXTextEdit2Script07, cases, t)
}

func TestCompareUsXTextEdit2Script(t *testing.T) {
	var tests []CtlTest
	tests = append(tests, CtlTest{BibleId: "ATIWBT", Expected: 8254, TextNtId: "ATIWBTN_ET-usx",
		TextType: request.TextUSXEdit, Language: "ati"})
	tests = append(tests, CtlTest{BibleId: "BGGWFW", Expected: 8765, TextNtId: "BGGWFWN_ET-usx",
		TextType: request.TextUSXEdit, Language: "bgg"})
	DirectTestUtility(TextUSXEdit07, tests, t)
	tests = nil
	tests = append(tests, CtlTest{BibleId: "ATIWBT", Expected: 9748, TextNtId: "ATIWBTN2ST",
		TextType: request.TextScript, Language: "ati"})
	DirectTestUtility(TextScript07, tests, t)
	tests = nil
	tests = append(tests, CtlTest{BibleId: "BGGWFW", Expected: 7881, TextNtId: "BGGWFWN2ST",
		TextType: request.TextScript, Language: "bgg"})
	DirectTestUtility(TextScript07BGGWFW, tests, t)
	tests = nil
	tests = append(tests, CtlTest{BibleId: "ATIWBT", Expected: 2, TextNtId: "ATIWBTN2ST",
		TextType: request.TextScript, Language: "ati"})
	tests = append(tests, CtlTest{BibleId: "BGGWFW", Expected: 1019, TextNtId: "BGGWFWN2ST",
		TextType: request.TextScript, Language: "bgg"})
	DirectTestUtility(CompareUsXTextEdit2Script07, tests, t)
}
