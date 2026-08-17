package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const zipFile = `is_new: yes
dataset_name: 03a_zip_file
bible_id: ENGWEB
username: Tests
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
audio_data:
  aws_s3: s3://arti-input/03a_zip_file_test_data/ENGWEBN2DA.zip
text_data:
  aws_s3: s3://arti-input/03a_zip_file_test_data/ENGWEB-usx.zip
testament:
  nt_books: [MRK]
`

func TestZipFileDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 694})
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts WHERE script_begin_ts != 0.0", 0})
	DirectSqlTest(zipFile, tests, t)
}
