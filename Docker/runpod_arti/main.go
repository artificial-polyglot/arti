package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/controller"
	log "github.com/artificial-polyglot/arti/logger"
)

func main() {
	status := run(os.Args[1:])
	if status != nil {
		os.Exit(1)
	}
}

func run(args []string) *log.Status {
	var ctx = context.WithValue(context.Background(), "runType", "runpod")
	if len(args) != 1 {
		return log.ErrorNoErr(ctx, 500, "usage: runpod_arti <request.yaml>")
	}
	yamlString := args[0]
	yamlBytes := []byte(yamlString)
	var control = controller.NewController(ctx, yamlBytes)
	_, status := control.ProcessV2()
	return status
}
