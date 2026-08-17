package tests

import (
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const plainTextEditScript = `is_new: yes
dataset_name: 01b_plain_text_edit_{bibleId}
bible_id: {bibleId}
username: Tests
notify_err: [ntfy/arti2]
testament:
  nt: yes
output:
  directory: ~/Downloads
  csv: yes
text_data:
  bible_brain:
    text_plain_edit: yes
detail:
  words: yes
`

func TestPlainTextEditDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 8218})
	testName := strings.Replace(plainTextEditScript, "{bibleId}", "ENGWEB", -1)
	DirectSqlTest(testName, tests, t)
}
