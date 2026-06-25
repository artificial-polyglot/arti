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
	dbPath := "/Users/gary/FCBH2024/GaryNTest/PlainTextEditScript_ENGWEB.db"
	r, w, _ := os.Pipe()
	_, _ = w.WriteString(dbPath + "\n")
	_ = w.Close()
	os.Stdin = r

	outR, outW, _ := os.Pipe()
	os.Stdout = outW

	// Call main
	main()

	// Read the output
	_ = outW.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(outR)
	output := strings.TrimRight(buf.String(), "\n")

	t.Logf("output: %s", output)
}
