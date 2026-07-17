package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/cmd/speech_to_text/qa_align"
	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

func main() {
	status := run(os.Args[1:])
	if status != nil {
		os.Exit(1)
	}
}

func run(args []string) *log.Status {
	var ctx = context.WithValue(context.Background(), "runType", "qa_align")
	if len(args) != 1 {
		return log.ErrorNoErr(ctx, 500, "usage: qa_align <request.yaml>")
	}
	yamlContent := args[0]
	component := courier.NewComponent(ctx, yamlContent, "qa_align")
	database, status := component.StartComponent()
	if status != nil {
		return status
	}
	defer database.Close()
	output, status := qa_align.Process(database)
	component.FinishComponent(output, status)
	if status != nil {
		return status
	}
	return nil
}
