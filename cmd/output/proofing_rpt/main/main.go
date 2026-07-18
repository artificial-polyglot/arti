package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
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
		return log.ErrorNoErr(context.Background(), 500, "usage: proofing_rpt <request.yaml>")
	}
	yamlContent := args[0]
	component := courier.NewComponent(yamlContent, "proofing_rpt")
	database, status := component.StartComponent()
	if status != nil {
		return status
	}
	defer database.Close()
	output, status := proofing_rpt.Process(database)
	if status != nil {
		return status
	}
	if len(output) > 0 {
		log.Info(context.Background(), output[0])
	}
	component.FinishComponent(output, status)
	return nil
}
