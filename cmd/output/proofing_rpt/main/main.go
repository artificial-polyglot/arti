package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
)

func main() {
	var ctx = context.WithValue(context.Background(), "runType", "proofing_rpt")
	if len(os.Args) != 2 {
		log.Fatal(ctx, "usage: proofing_rpt <request.yaml>")
	}
	yamlContent := os.Args[1]
	component := courier.NewComponent(ctx, yamlContent, "proofing_rpt")
	database, status := component.StartComponent()
	if status != nil {
		os.Exit(1)
	}
	defer database.Close()
	output, status := proofing_rpt.Process(database)
	if status != nil {
		os.Exit(1)
	}
	component.FinishComponent(output, status)
}
