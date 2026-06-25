package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/stdio_exec"
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
