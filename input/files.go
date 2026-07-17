package input

import (
	"context"

	log "github.com/artificial-polyglot/arti/logger"
)

func FileInput(ctx context.Context, path string) ([]InputFile, *log.Status) {
	var files []InputFile
	var status *log.Status
	files, status = Glob(ctx, path)
	for i := range files {
		files[i].BaseURL = "file://" + files[i].Directory
	}
	return files, status
}
