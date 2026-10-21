package main

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

const compare_rpt_yaml = `
is_new: false 
dataset_name: N2CCPBBS
username: GaryNTest
language_iso: ccp
database:
    aws_s3: s3://arti-output/GaryNTest/N2CCPBBS/arti/00025/database/N2CCPBBS.db
compare:
    html_report: true
`

func TestRun(t *testing.T) {
	log.SetOutput("stderr")
	courier.IsCourierTest = false
	status := run([]string{compare_rpt_yaml})
	if status != nil {
		t.Error(status)
		t.Fatal(status)
	}
}
