package main

import (
	"fmt"
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
	if len(args) < 1 {
		_, _ = fmt.Fprintln(os.Stdout, "Usage: dataset  request.yaml")
		os.Exit(1)
	}
	var content, err = os.ReadFile(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, "Error reading yaml request file.")
		os.Exit(1)
	}
	outputFile, status := controller.CLIProcessEntry(content)
	if status != nil {
		_, _ = fmt.Fprintln(os.Stderr, status.String())
		os.Exit(1)
	} else {
		for _, file := range outputFile.FilePaths {
			_, _ = fmt.Fprintln(os.Stdout, `Success:`, file)
		}
	}
	return nil
}
