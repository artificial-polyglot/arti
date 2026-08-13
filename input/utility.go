package input

import (
	"context"
	"path/filepath"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
)

func FileInput(ctx context.Context, path string) ([]generic.InputFile, *log.Status) {
	var files []generic.InputFile
	var status *log.Status
	files, status = Glob(ctx, path)
	for i := range files {
		files[i].BaseURL = "file://" + files[i].Directory
	}
	return files, status
}

func Glob(ctx context.Context, search string) ([]generic.InputFile, *log.Status) {
	var results []generic.InputFile
	if search != `` {
		files, err := filepath.Glob(search)
		if err != nil {
			return results, log.Error(ctx, 500, err, "Error expanding file glob pattern:", search)
		}
		for _, file := range files {
			var inp generic.InputFile
			inp.Directory = filepath.Dir(file)
			inp.Filename = filepath.Base(file)
			results = append(results, inp)
		}
	}
	return results, nil
}

// UpdateIdent, updateIdentText, and updateIdentAudio moved to the db
// package (db/audio_files.go) since they touch the SQL database - keeping
// them here pulled modernc.org/sqlite into any build (like the WASM
// validator) that only needs the InputFile type.

// Parse DBP4 Audio names
//{mediaid}_{A/B}{ordering}_{USFM book code}_{chapter start}[_{verse start}-{chapter stop}_{verse stop}].mp3|webm
//eg: ENGESVN2DA_B001_MAT_001.mp3  (full chapter)
//eg: IRUNLCP1DA_B013_1TH_001_001-001_010.mp3  (partial chapter, verses 1-10)
