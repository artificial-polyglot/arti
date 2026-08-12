package input

import (
	"context"
	"path/filepath"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
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

func Glob(ctx context.Context, search string) ([]InputFile, *log.Status) {
	var results []InputFile
	if search != `` {
		files, err := filepath.Glob(search)
		if err != nil {
			return results, log.Error(ctx, 500, err, "Error expanding file glob pattern:", search)
		}
		for _, file := range files {
			var inp InputFile
			inp.Directory = filepath.Dir(file)
			inp.Filename = filepath.Base(file)
			results = append(results, inp)
		}
	}
	return results, nil
}

func UpdateIdent(conn db.DBAdapter, ident *db.Ident, textFiles []InputFile, audioFiles []InputFile) *log.Status {
	var status *log.Status
	_ = updateIdentText(ident, textFiles)
	_ = updateIdentAudio(ident, audioFiles)
	status = conn.InsertReplaceIdent(*ident)
	return status
}

func updateIdentText(ident *db.Ident, files []InputFile) bool {
	var result = false
	if len(files) > 0 {
		f := files[0]
		if f.MediaType != request.TextNone {
			ident.TextSource = f.MediaType
		}
		result = true
		if f.Testament == `OT` {
			ident.TextOTId = f.MediaId
		} else if f.Testament == `NT` {
			ident.TextNTId = f.MediaId
		}
	}
	return result
}

func updateIdentAudio(ident *db.Ident, files []InputFile) bool {
	var result = false
	for _, f := range files {
		result = true
		if f.Testament == `OT` {
			ident.AudioOTId = f.MediaId
		} else if f.Testament == `NT` {
			ident.AudioNTId = f.MediaId
		}
	}
	return result
}

// Parse DBP4 Audio names
//{mediaid}_{A/B}{ordering}_{USFM book code}_{chapter start}[_{verse start}-{chapter stop}_{verse stop}].mp3|webm
//eg: ENGESVN2DA_B001_MAT_001.mp3  (full chapter)
//eg: IRUNLCP1DA_B013_1TH_001_001-001_010.mp3  (partial chapter, verses 1-10)
