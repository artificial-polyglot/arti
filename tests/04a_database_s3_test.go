package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const databaseS3 = `is_new: no
dataset_name: 04a_database_s3_ENGWEB
bible_id: ENGWEB
username: Tests
output:
  directory: ~/Downloads
  csv: yes
database:
  aws_s3: s3://arti-output/GaryNTest/N2ATGMLT/arti/00001/database/N2ATGMLT.db
`

// Prior test
//   aws_s3: s3://dataset-io/GaryNTest/01a_plain_text_ENGWEB/00004/database/01a_plain_text_ENGWEB.db

func TestDatabaseS3Direct(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 8219})
	DirectSqlTest(databaseS3, tests, t)
}
