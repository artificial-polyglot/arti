package tests

import "testing"

const scriptVersifierScript = `is_new: yes
dataset_name: 3d_script_versifier
username: GaryNTest
language_iso: ako
output:
  sqlite: yes
text_data:
  file: /Users/gary/FCBH2024/GaryNTest/3d_script_versifier/Text_Aokho_N2IKHMLT.xlsx
detail:
  verses: true
`

func TestScriptVersifier(t *testing.T) {
	var tests1 []SqliteTest
	tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM scripts", 8220})
	//tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM words where ttype='W'", 574228})
	_ = DirectSqlTest(scriptVersifierScript, tests1, t)
}
