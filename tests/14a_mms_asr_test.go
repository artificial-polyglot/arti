package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

const mMSASRTest = `is_new: yes
dataset_name: 14a_mms_asr
bible_id: ENGWEB
username: Tests
output:
  directory: ~/Downloads
  csv: yes
text_data:
  bible_brain:
    text_plain_edit: yes
audio_data:
  bible_brain:
    mp3_64: yes
timestamps:
  mms_align:
  bible_brain: yes
testament:
  nt_books: [PHM]
speech_to_text:
  mms_asr: yes
stt_decoder:
  greedy: yes
  simple: no
  hotwords: no
  kenlm: no
`

func TestMMSASRDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 26})
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts WHERE script_begin_ts != 0.0", 25})
	log.SetOutput("stderr")
	DirectSqlTest(mMSASRTest, tests, t)
}
