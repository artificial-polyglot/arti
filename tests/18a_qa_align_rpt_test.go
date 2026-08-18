package tests

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

// This is not a test that is expected to run to completion.
// It exists so that one can debug the startup of inference

const qaAlignRpt = `is_new: no
dataset_name: 18a_qa_align_rpt_test
username: Tests
language_iso: atg # qae works
notify_err: [ntfy/arti2]
output:
  directory: ~/Downloads
  csv: yes
testament:
  nt_books: [3JN] # 1JN, 2JN, 3JN, JUD all in test set
database:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/17a_mms_adapter.db
audio_data:
  aws_s3: s3://arti-input/17a_mms_adapter_test_data/*.mp3
audio_proof:
  html_report: yes
`

func TestQAAlignTest(t *testing.T) {
	courier.IsCourierTest = true
	var yaml = qaAlignRpt
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
