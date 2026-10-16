package main

import (
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

const train_yaml = "/Users/gary/arti2/N2TTS_test2.yaml"

func TestRun(t *testing.T) {
	log.SetOutput("stderr")
	courier.IsCourierTest = false
	status := run([]string{train_yaml})
	if status != nil {
		t.Error(status)
		t.Fatal(status)
	}
}
