package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	log "github.com/artificial-polyglot/arti/logger"
)

func TestRun(t *testing.T) {
	log.SetOutput("stderr")
	file := "/Users/gary/arti2/N2TTS.yaml"
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	status := run([]string{string(content)})
	if status != nil {
		t.Error(status)
		t.Fatal(status)
	}
	t.Logf("output: %s", strings.TrimRight(out.String(), "\n"))
}
