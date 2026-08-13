package input

import (
	"context"
	"strings"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

// InputFile and its FilePath() method moved to the generic package, so that
// input/precheck (and thus the WASM validator) can use the type without
// pulling in input's own dependencies (s3_datastore/AWS SDK, and formerly
// db - see db/audio_files.go). Download stays here as a plain function
// since a method's receiver type must live in the same package as the
// method, and generic is meant to stay dependency-free.
func Download(file *generic.InputFile, ctx context.Context, client s3_datastore.S3Client, tempDir string) *log.Status {
	var status *log.Status
	if strings.HasPrefix(file.BaseURL, "file://") {
		file.Directory = file.BaseURL[7:]
		return nil
	}
	if strings.HasPrefix(file.BaseURL, "s3://") {
		s3Path := file.BaseURL[5:]
		slash := strings.Index(s3Path, "/")
		bucket := s3Path[0:slash]
		prefix := s3Path[slash+1:]
		var objectKey string
		if strings.HasSuffix(prefix, "/") {
			objectKey = prefix + file.Filename
		} else {
			objectKey = prefix + "/" + file.Filename
		}
		file.Directory = tempDir
		status = client.DownloadFile(bucket, objectKey, file.FilePath())
		return status
	} else {
		return log.ErrorNoErr(ctx, 500, "InputFile.Download supports s3://, and file://")
	}
}

// InsertAudioFiles, SelectAudioFiles, and SelectBaseURL moved to the db
// package (db/audio_files.go) since they touch the SQL database - keeping
// them here pulled modernc.org/sqlite into any build (like the WASM
// validator) that only needs the InputFile type.
