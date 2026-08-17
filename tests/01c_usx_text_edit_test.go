package tests

import (
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const uSXTextEditScript = `is_new: yes
dataset_name: 01c_usx_text_edit_{bibleId}
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
    text_usx_edit: yes
detail:
  words: yes
`

func TestUSXTextEditDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 8213})
	testName := strings.Replace(uSXTextEditScript, "{bibleId}", "ENGWEB", -1)
	DirectSqlTest(testName, tests, t)
}
