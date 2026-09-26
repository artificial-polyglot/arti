package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/cmd/output/compare_rpt"
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
		return log.ErrorNoErr(context.Background(), 500, "usage: compare_rpt <request.yaml>")
	}
	yamlContent := args[0]
	component := courier.NewComponent(yamlContent, "compare_rpt")
	database, request, status := component.StartComponent()
	if status != nil {
		return status
	}
	defer database.Close()
	output, status := compare_rpt.Process(database, request)
	if status != nil {
		return status
	}
	if len(output) > 0 {
		log.Info(context.Background(), output[0])
	}
	component.FinishComponent(output, status)
	return nil
}
