package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
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

	// get uroman from request
	rpt := proofing_rpt.NewProofingRpt(ctx, conn, req.LanguageISO, false)
	records, verses, status := rpt.Process()
	if status != nil {
		goodbye(status)
	}

	writer := proofing_rpt.NewHTMLWriter(ctx, conn.Project)
	filename, status := writer.WriteReport(records, verses, req.LanguageISO, req.SpeechToText)
	if status != nil {
		goodbye(status)
	}
	status = conn.InsertOutput("proofing_rpt", "proofing", filename)
	if status != nil {
		goodbye(status)
	}
	fmt.Println(dbPathname)
}

func goodbye(status *log.Status) {
	_, _ = fmt.Fprintln(os.Stderr, status)
	os.Exit(1)
}
