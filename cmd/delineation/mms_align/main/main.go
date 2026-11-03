package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/cmd/delineation/mms_align"
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
	if len(args) != 1 {
		return log.ErrorNoErr(context.Background(), 500, "usage: mms_align <request.yaml>")
	}
	yamlContent := args[0]
	component := courier.NewComponent(yamlContent, "mms_align")
	database, request, status := component.StartComponent()
	if status != nil {
		return status
	}
	defer database.Close()
	output, status := mms_align.Process(database, request)
	if status != nil {
		return status
	}
	for _, out := range output {
		log.Info(context.Background(), out.Component, out.Report, out.FilePath)
	}
	component.FinishComponent(output, status)
	return nil
}
