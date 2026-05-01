package tests

import (
	"testing"
)

const usfmVsUSX = `is_new: yes
dataset_name: 3c_usfm_vs_usx_O2
bible_id: O2NHEWYI
username: GaryNTest
output:
  sqlite: yes
text_data:
  aws_s3: s3://pretest-audio/O2NHEWYI Nahuatl Huasteca, Eastern (NHE)/O2NHEWYI Text/O2NHEWYI SFM/*.SFM
detail:
  words: yes
`

// This is not yet testing usfm vs usx

func TestUSFMReadDirect(t *testing.T) {
	var tests1 []SqliteTest
	tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM scripts", 23790})
	tests1 = append(tests1, SqliteTest{"SELECT count(*) FROM words where ttype='W'", 574228})
	_ = DirectSqlTest(usfmVsUSX, tests1, t)
}
