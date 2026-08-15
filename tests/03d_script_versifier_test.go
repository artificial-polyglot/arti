package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const scriptVersifierScript = `is_new: yes
dataset_name: 03d_script_versifier
username: Tests
language_iso: ako
output:
  directory: ~/Downloads
  csv: yes
text_data:
  file: 03d_script_versifier_test_data/Text_Aokho_N2IKHMLT.xlsx
detail:
  verses: true
`

func TestScriptVersifier(t *testing.T) {
	courier.IsCourierTest = true
	var tests1 []SqliteTest
	tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM scripts", 8220})
	//tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM words where ttype='W'", 574228})
	_ = DirectSqlTest(scriptVersifierScript, tests1, t)
}
