package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/controller"
	log "github.com/artificial-polyglot/arti/logger"
)

func main() {
	var ctx = context.WithValue(context.Background(), "runType", "runpod")
	if len(os.Args) != 2 {
		log.Fatal(ctx, "usage: runpod_arti <request.yaml>")
	}
	yamlString := os.Args[1]
	yamlBytes := []byte(yamlString)
	var control = controller.NewController(ctx, yamlBytes)
	_, status := control.ProcessV2()
	if status != nil {
		os.Exit(1)
	}
}
