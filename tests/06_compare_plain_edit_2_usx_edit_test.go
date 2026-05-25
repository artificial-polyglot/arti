package tests

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"strings"
	"testing"
)

const ComparePlainEdit2USXEditScript = `is_new: no
dataset_name: PlainTextEditScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
compare:
  html_report: yes
  base_dataset: USX Text Edit Script_{bibleId}
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

func TestComparePlainEdit2USXEditScriptAPI(t *testing.T) {
	var cases []APITest
	cases = append(cases, APITest{BibleId: `ENGWEB`, Expected: 0})
	APITestUtility(ComparePlainEdit2USXEditScript, cases, t)
}

func TestComparePlainEdit2USXEditScript(t *testing.T) {
	var bibleId = `ENGWEB`
	ctx := context.Background()
	var req = strings.Replace(ComparePlainEdit2USXEditScript, `{bibleId}`, bibleId, 3)
	var control = controller.NewController(ctx, []byte(req))
	filename, status := control.Process()
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("Filename", filename)
	count := NumHTMLFileLines(filename, t)
	expected := 0
	if count != expected {
		t.Error(`expected`, expected, `found`, count)
	}
	identTest(`PlainTextEditScript_`+bibleId, t, request.TextPlainEdit, ``,
		`ENGWEBN_ET`, ``, ``, `eng`)
}
