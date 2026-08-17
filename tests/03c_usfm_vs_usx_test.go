package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const usfmVsUSX = `is_new: yes
dataset_name: 03c_usfm_vs_usx_O2
language_iso: qae
username: Tests
output:
  directory: ~/Downloads
  csv: yes
text_data:
  aws_s3: s3://arti-input/N2QAEBSP/Text Files/SFM Text/*.SFM
detail:
  words: yes
`

// This is not yet testing usfm vs usx

func TestUSFMReadDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests1 []SqliteTest
	tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM scripts", 3866})
	tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM words where ttype='W'", 101151})
	_ = DirectSqlTest(usfmVsUSX, tests1, t)
}
