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
	content, err := os.ReadFile("/Users/gary/arti2/N2CCPBBS_qa.yaml")
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
