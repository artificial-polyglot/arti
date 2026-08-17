package tests

import (
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const scriptTextScript = `is_new: yes
dataset_name: 01d_script_text_{bibleId}
bible_id: {bibleId}
username: Tests
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
text_data:
  file: ../read/script_reader/test_data/CORE_Scr_1065p_1Eng__14_Spkr_Tajik_N2_TGK_IBT_arti.xlsx
`

func TestScriptTextDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 7668})
	testName := strings.Replace(scriptTextScript, "{bibleId}", "TGKIBT", -1)
	DirectSqlTest(testName, tests, t)
}
