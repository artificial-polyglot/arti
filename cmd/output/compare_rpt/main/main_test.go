package main

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

const compare_rpt_yaml = `
is_new: false 
dataset_name: N2TTS
username: GaryNTest
language_iso: tts
database:
    aws_s3: s3://arti-output/GaryNTest/N2TTS/arti/00041/database/N2TTS.db
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
