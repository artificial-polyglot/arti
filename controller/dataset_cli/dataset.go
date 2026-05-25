package main

import (
	"fmt"
	"github.com/artificial-polyglot/arti/controller"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintln(os.Stdout, "Usage: dataset  request.yaml")
		os.Exit(1)
	}
	var content, err = os.ReadFile(os.Args[1])
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
}
