package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

func TestRun(t *testing.T) {
	log.SetOutput("stderr")
	courier.IsCourierTest = true
	content, err := os.ReadFile("/Users/gary/arti2/N2ATGMLT_rpt.yaml")
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
