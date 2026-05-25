package tests

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"strings"
	"testing"
)

const AudioWhisperJson = `is_new: yes
dataset_name: AudioWhisperJson_{bibleId}
bible_id: {bibleId}
username: GaryNTest
email: gary@shortsands.com
output:
  json: yes
testament: # Choose one or both
  nt_books: [PHM]
text_data:
  bible_brain:
    text_usx_edit: yes
audio_data:
  bible_brain:
    mp3_64: yes
timestamps:
  ts_bucket: yes
speech_to_text:
  whisper:
    model:
      tiny: yes
`

func TestAudioWhisperJsonAPI(t *testing.T) {
	var cases []APITest
	cases = append(cases, APITest{BibleId: `ENGWEB`, Expected: 35, Diff: 5})
	APITestUtility(AudioWhisperJson, cases, t)
}

func TestAudioWhisperJson(t *testing.T) {
	var bibles = make(map[string]int)
	bibles[`ENGWEB`] = 35
	ctx := context.Background()
	for bibleId, expected := range bibles {
		var req = strings.Replace(AudioWhisperJson, `{bibleId}`, bibleId, 2)
		ctrl := controller.NewController(ctx, []byte(req))
		filename, status := ctrl.Process()
		fmt.Println("Filename", filename, status)
		if status != nil {
			t.Fatal(status)
		}
		numLines := NumJSONFileLines(filename, t)
		if numLines != expected {
			t.Error(`Expected `, expected, `records, got`, numLines)
		}
		identTest(`AudioWhisperJson_`+bibleId, t, request.TextSTT, ``,
			`ENGWEBN_TT`, ``, `ENGWEBN2DA`, `eng`)
	}
}

func TestAudioWhisperJsonCLI(t *testing.T) {
	var bibles = make(map[string]int)
	bibles[`ENGWEB`] = 26
	for bibleId, expected := range bibles {
		var request = strings.Replace(AudioWhisperJson, `{bibleId}`, bibleId, 2)
		stdout, stderr := CLIExec(request, t)
		fmt.Println(`STDOUT:`, stdout)
		fmt.Println(`STDERR:`, stderr)
		filename := ExtractFilename(request)
		numLines := NumJSONFileLines(filename, t)
		if numLines != expected {
			t.Error(`Expected `, expected, `records, got`, numLines)
		}
	}
}
