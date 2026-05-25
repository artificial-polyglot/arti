package tests

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"strings"
	"testing"
)

const ComparePlain2PlainEditScript = `is_new: no
dataset_name: PlainTextScript_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
compare:
  html_report: yes
  base_dataset: PlainTextEditScript_{bibleId}
  compare_settings: # Mark yes, all settings that apply
    lower_case: 
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

func TestComparePlain2PlainEditScriptAPI(t *testing.T) {
	var cases []APITest
	cases = append(cases, APITest{BibleId: `ENGWEB`, Expected: 260})
	APITestUtility(ComparePlain2PlainEditScript, cases, t)
}

func TestComparePlain2PlainEditScriptCLI(t *testing.T) {
	var bibleId = `ENGWEB`
	var req = strings.Replace(ComparePlain2PlainEditScript, `{bibleId}`, bibleId, 3)
	stdout, stderr := CLIExec(req, t)
	fmt.Println(`STDOUT:`, stdout)
	fmt.Println(`STDERR:`, stderr)
	filename := ExtractFilename(stdout)
	count := NumHTMLFileLines(filename, t)
	expected := 260
	if count != expected {
		t.Error(`expected`, expected, `found`, count)
	}
}

func TestComparePlain2PlainEditScript(t *testing.T) {
	var bibleId = `ENGWEB`
	ctx := context.Background()
	var req = strings.Replace(ComparePlain2PlainEditScript, `{bibleId}`, bibleId, 3)
	var control = controller.NewController(ctx, []byte(req))
	filename, status := control.Process()
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("Filename", filename)
	count := NumHTMLFileLines(filename, t)
	expected := 276
	if count != expected {
		t.Error(`expected`, expected, `found`, count)
	}
	identTest(`PlainTextScript_`+bibleId, t, request.TextPlain, ``,
		`ENGWEBN_ET`, ``, ``, `eng`)
}
