package input

import (
	"context"
	"os"
	"path/filepath"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

// DBPDirectory 1. Assign pattern for OT, NT.  2. Glob files.  3. Assign book/chapter & Prune
func DBPDirectory(ctx context.Context, bibleId string, fsType request.MediaType, otFileset string,
	ntFileset string) ([]generic.InputFile, *log.Status) {
	var results []generic.InputFile
	var files []generic.InputFile
	var status *log.Status
	type run struct {
		filesetId string
		tType     string
	}
	var runs []run
	if otFileset != `` {
		runs = append(runs, run{filesetId: otFileset, tType: `OT`})
	}
	if ntFileset != `` {
		runs = append(runs, run{filesetId: ntFileset, tType: `NT`})
	}
	for _, r := range runs {
		files, status = Directory(ctx, bibleId, fsType, r.filesetId, r.tType)
		if status != nil {
			return results, status
		}
		results = append(results, files...)
	}
	return results, status
}

func Directory(ctx context.Context, bibleId string, fsType request.MediaType, filesetId string, tType string) ([]generic.InputFile, *log.Status) {
	var status *log.Status
	var directory string
	var search string
	if fsType == request.TextPlain || fsType == request.TextPlainEdit {
		directory = filepath.Join(os.Getenv("FCBH_DATASET_FILES"), bibleId)
		search = filepath.Join(directory, filesetId+".json")
	} else if fsType == request.TextUSXEdit {
		directory = filepath.Join(os.Getenv("FCBH_DATASET_FILES"), bibleId, filesetId)
		search = filepath.Join(directory, "*.usx")
	} else if fsType == request.Audio || fsType == request.AudioDrama {
		directory = filepath.Join(os.Getenv("FCBH_DATASET_FILES"), bibleId, filesetId)
		if tType == `OT` {
			search = filepath.Join(directory, "A*.mp3")
		} else {
			search = filepath.Join(directory, "B*.mp3")
		}
	}
	//fmt.Println("search:", tType, search)
	var files []generic.InputFile
	files, status = Glob(ctx, search)
	for i := range files {
		files[i].BaseURL = "file://" + directory
		files[i].MediaType = fsType
	}
	return files, status
}
