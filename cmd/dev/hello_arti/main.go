package main

import (
	"context"
	"os"

	"github.com/artificial-polyglot/arti/courier"
	"github.com/artificial-polyglot/arti/decode_yaml"
	log "github.com/artificial-polyglot/arti/logger"
)

/**
This is intended to the simplest arti-like program to be used for learning
how to package arti in docker, and deploy serverless on runpod,
so that it can be run anywhere.
*/

func main() {
	ctx := context.Background()
	log.SetOutput("stderr") // Should this be file for more valid text
	if len(os.Args) != 2 {
		log.Fatal(ctx, "usage: hello_arti <request.yaml>")
	}
	log.Info(ctx, "Have the required parameter")
	yamlString := os.Args[1]
	yamlBytes := []byte(yamlString)
	bucket := courier.NewCourier(ctx, yamlBytes)
	log.Info(ctx, "Courier instantiated")
	reqDecoder := decode_yaml.NewRequestDecoder(ctx)
	_, status := reqDecoder.Process(yamlBytes)
	if status != nil {
		os.Exit(1)
	}
	log.Info(ctx, "Yaml decoded")
	status = bucket.PersistToBucket()
	log.Info(ctx, "Finished status=", status)
}
