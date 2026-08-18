package input

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

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

// DownloadFile is used by Controller to download a database file
func DownloadDatabaseFile(ctx context.Context, s3Path string, filePath string) *log.Status {
	err := os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return log.Error(ctx, 500, err, "Attempt to overwrite a file with a directory.")
	}
	client, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		return status
	}
	bucket, objectKey, _, status := parseGlob(ctx, s3Path)
	if status != nil {
		return status
	}
	log.Info(ctx, `Downloading file`, objectKey)
	response, getErr := client.GetObject(bucket, objectKey)
	if getErr != nil {
		return log.Error(ctx, 400, getErr, `Failed to get S3 object`, objectKey)
	}
	filErr := os.WriteFile(filePath, response, 0644)
	if filErr != nil {
		return log.Error(ctx, 400, filErr, `Failed to create file of S3 object`, filePath)
	}
	return nil
}

func DownloadToDBLocation(ctx context.Context, dbPath string, username string) (string, *log.Status) {
	baseName := filepath.Base(dbPath)
	fullPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), username, baseName)
	var err error
	if strings.HasPrefix(dbPath, "file://") {
		err = os.Rename(dbPath[:7], fullPath)
	} else if strings.HasPrefix(dbPath, "s3://") {
		status := DownloadDatabaseFile(ctx, dbPath, fullPath)
		if status != nil {
			return baseName, status
		}
	} else if strings.Contains(dbPath, "/") {
		err = os.Rename(dbPath, fullPath)
	}
	if err != nil {
		return baseName, log.Error(ctx, 500, err, "Could not rename file to correct DB location")
	}
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	return baseName, nil
}
