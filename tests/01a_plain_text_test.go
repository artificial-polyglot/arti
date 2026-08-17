package tests

import (
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const plainTextScript = `is_new: yes
dataset_name: 01a_plain_text_{bibleId}
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
    text_plain: yes
detail:
  words: yes
`

func TestPlainTextDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 7958})
	testName := strings.Replace(plainTextScript, "{bibleId}", "ENGWEB", -1)
	DirectSqlTest(testName, tests, t)
}
