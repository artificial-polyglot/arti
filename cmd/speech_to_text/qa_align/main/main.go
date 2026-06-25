package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/artificial-polyglot/arti/cmd/speech_to_text/qa_align"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func main() {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)
	dbPathname, _ := reader.ReadString('\n')
	conn := db.NewDBAdapter(ctx, strings.TrimSpace(dbPathname))
	req, status := conn.SelectRequest()
	if status != nil {
		goodbye(status)
	}

	// how is adapter set from req
	asr := qa_align.NewQAAlign(ctx, conn, req.LanguageISO, req.AltLanguage, false)
	status = asr.ProcessFiles()
	if status != nil {
		goodbye(status)
	}
	fmt.Println(dbPathname)
}

func goodbye(status *log.Status) {
	_, _ = fmt.Fprintln(os.Stderr, status)
	os.Exit(1)
}
