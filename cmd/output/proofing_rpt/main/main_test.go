package main

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

const proofing_rpt_yaml = `
is_new: false 
dataset_name: N2CCPBBS
username: GaryNTest
language_iso: ccp
database:
    aws_s3: s3://arti-output/GaryNTest/N2CCPBBS/arti/00016/database/N2CCPBBS.db
audio_proof:
    html_report: true
`

func TestRun(t *testing.T) {
	log.SetOutput("stderr")
	courier.IsCourierTest = false
	status := run([]string{proofing_rpt_yaml})
	if status != nil {
		t.Error(status)
		t.Fatal(status)
	}
}
