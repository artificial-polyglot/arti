package tests

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"strings"
	"testing"
)

const ComparePlainTextEdit2Script = `is_new: no
dataset_name: ScriptTextScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
compare:
  html_report: yes
  base_dataset: PlainTextEditScript_{bibleId}
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

// Danger: This test is dependent on test 4 to create the script file.
func TestComparePlainTextEdit2ScriptAPI(t *testing.T) {
	var cases []APITest
	cases = append(cases, APITest{BibleId: `ATIWBT`, Expected: 2})
	APITestUtility(ComparePlainTextEdit2Script, cases, t)
}

func TestComparePlainTextEdit2Script(t *testing.T) {
	var bibleId = `ATIWBT`
	ctx := context.Background()
	var req = strings.Replace(ComparePlainTextEdit2Script, `{bibleId}`, bibleId, 3)
	var control = controller.NewController(ctx, []byte(req))
	filename, status := control.Process()
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("Filename", filename)
	count := NumHTMLFileLines(filename, t)
	expected := 2
	if count != expected {
		t.Error(`expected`, expected, `found`, count)
	}
	identTest(`ScriptTextScript_`+bibleId, t, request.TextScript, ``,
		`ATIWBTN2ST`, ``, ``, `ati`)
}
