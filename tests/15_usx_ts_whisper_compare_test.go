package tests

import (
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"testing"
)

const USXTSWhisperCompare = `is_new: yes
dataset_name: USXTSWhisperCompare_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output_file: 15__usx_ts_whisper_compare.html
text_data:
  bible_brain:
    text_usx_edit: yes
audio_data:
  bible_brain:
    mp3_64: yes
timestamps:
  aeneas: yes
testament:
  nt_books: ['3JN']
speech_to_text:
  whisper:
    model:
      tiny: yes
compare:
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

func TestUSXTSWhisperCompare(t *testing.T) {
	var tests []CtlTest
	//tests = append(tests, CtlTest{BibleId: "ENGWEB", Expected: 27, TextNtId: "ENGWEBN_ET-usx",
	//	TextType: request.TextUSXEdit, AudioNTId: "ENGWEBN2DA-mp3-16", Language: "eng"})
	tests = append(tests, CtlTest{BibleId: "APFCMU", Expected: 16, TextNtId: "APFCMUN_ET-usx",
		AudioNTId: `APFCMUN1DA`, TextType: request.TextUSXEdit, Language: "apf"})
	//tests = append(tests, try{bibleId: "ATIWBT", expected: 7, textNtId: "ATIWBTN_ET", audioNTId: "ATIWBTN1DA",
	//	language: "ati"}) // There are no timestamps
	DirectTestUtility(USXTSWhisperCompare, tests, t)
}

func TestPlainWhisperCompare(t *testing.T) {
	const PlainTSWhisperCompare = `is_new: yes
dataset_name: PlainTSWhisperCompare_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output_file: 15__plain_ts_whisper_compare.html
text_data:
  bible_brain:
    text_plain_edit: yes
audio_data:
  bible_brain:
    mp3_64: yes
timestamps:
  aeneas: yes
testament:
  nt_books: ['TIT']
compare:
  base_dataset: USXTSWhisperCompare_{bibleId}_STT
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
	var tests []CtlTest
	tests = append(tests, CtlTest{BibleId: "ENGWEB", Expected: 27, TextNtId: "ENGWEBN_ET",
		AudioNTId: `ENGWEBN2DA`, TextType: request.TextPlainEdit, Language: "eng"})
	//tests = append(tests, try{bibleId: "ATIWBT", expected: 7, textNtId: "ATIWBTN_ET", audioNTId: "ATIWBTN1DA",
	//	language: "ati"}) // There are no timestamps
	DirectTestUtility(PlainTSWhisperCompare, tests, t)
}

/*

AGNWPSN2DA agn {tgl tl PH} no USX
ALPWBTN1DA alp {ind id ID} no USX
AMKWBTN1DA amk {ind id ID} no USX text_plain fileset is AMKWBT
APFCMUN1DA apf {tgl tl PH}
BKVWYIN2DA bkv {hau ha NG}
BNOWBTN1DA bno {tgl tl PH}
BPSWPSN2DA bps {tgl tl PH}
CGCTBLN1DA cgc {tgl tl PH}
DSHBTLN1DA dsh {amh am ET}
DWRTBLN2DA dwr {amh am ET}
EKAWYIN1DA eka {hau ha NG}
ENGWEBN2DA eng {eng en English}
IFAWBTN1DA ifa {tgl tl PH}
IFBTBLN2DA ifb {tgl tl PH}
IFUWPSN2DA ifu {tgl tl PH}
IFYWBTN2DA ify {tgl tl PH}
IRIWYIN1DA iri {hau ha NG}
KCGWBTN1DA kcg {hau ha NG}
KNETBLN1DA kne {tgl tl PH}
KQETBLN1DA kqe {tgl tl PH}
LEXWBTN1DA lex {ind id ID}
MBTWBTN2DA mbt {tgl tl PH}
MNBTBLN2DA mnb {ind id ID}
MTJTBLN1DA mtj {ind id ID}
NINWYIN1DA nin {hau ha NG}
PCMTSCN2DA pcm {hau ha NG}
RMORAMN2DA rmo {deu de DE}
SGBTBLN2DA sgb {tgl tl PH}
*/
