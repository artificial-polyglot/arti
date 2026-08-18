package input

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// https://aws.github.io/aws-sdk-go-v2/docs/

// AWSS3Input is given a path prefix, that it uses to identify files.
// Saves each file found to disk, and returns an array of input files
func AWSS3Input(ctx context.Context, path string) ([]generic.InputFile, *log.Status) {
	var files []generic.InputFile
	var status *log.Status
	client, err := s3_datastore.NewS3Client(ctx)
	if err != nil {
		return files, log.Error(ctx, 400, err, "Failed to load S3 configuration")
	}
	bucket, prefix, glob, status := parseGlob(ctx, path)
	if status != nil {
		return files, status
	}
	list, err := client.ListObjects(bucket, prefix)
	if err != nil {
		return files, log.Error(ctx, 400, err, "Failed to list S3 objects at path:", path)
	}
	directory := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), prefix)
	status = EnsureDirectory(ctx, directory)
	for _, object := range list {
		objKey := aws.ToString(object.Key)
		if glob == nil || glob.MatchString(objKey) {
			var inFile generic.InputFile
			inFile.BaseURL = "s3://" + bucket + "/" + prefix
			inFile.Directory = directory
			inFile.Filename = filepath.Base(objKey)
			files = append(files, inFile)
			filePath := inFile.FilePath()
			fileInfo, stErr := os.Stat(filePath)
			if os.IsNotExist(stErr) || fileInfo.Size() != *object.Size {
				log.Info(ctx, `Downloading file`, objKey)
				response, getErr := client.GetObject(bucket, objKey)
				if getErr != nil {
					return files, log.Error(ctx, 400, getErr, `Failed to get object`, objKey)
				}
				filErr := os.WriteFile(filePath, response, 0644)
				if filErr != nil {
					return files, log.Error(ctx, 400, filErr, `Failed to create file`, filePath)
				}
			}
		}
	}
	return files, nil
}

func EnsureDirectory(ctx context.Context, directory string) *log.Status {
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		err2 := os.MkdirAll(directory, os.ModePerm)
		if err2 != nil {
			return log.Error(ctx, 400, err2, "Failed to create local directory for S3 downloads:", directory)
		}
	} else if err != nil {
		return log.Error(ctx, 400, err, "Failed to stat local directory:", directory)
	}
	return nil
}

func parseGlob(ctx context.Context, globKey string) (string, string, *regexp.Regexp, *log.Status) {
	var bucket string
	var prefix string
	var regex *regexp.Regexp
	var status *log.Status
	if strings.HasPrefix(globKey, `s3://`) {
		globKey = globKey[5:]
	} else if strings.HasPrefix(globKey, `s3:/`) {
		globKey = globKey[4:]
	}
	firstSlash := strings.Index(globKey, `/`)
	if firstSlash >= 0 {
		bucket = globKey[:firstSlash]
		prefix = globKey[firstSlash+1:]
		regex = nil
	}
	lastSlash := strings.LastIndex(globKey, `/`)
	if lastSlash >= 0 {
		glob := globKey[lastSlash+1:]
		if strings.Contains(glob, `*`) {
			prefix = globKey[firstSlash+1 : lastSlash+1]
			regex, status = globPattern(ctx, glob)
			if status != nil {
				return bucket, prefix, regex, status
			}
		}
	}
	return bucket, prefix, regex, status
}

func globPattern(ctx context.Context, glob string) (*regexp.Regexp, *log.Status) {
	var regex *regexp.Regexp
	var err error
	glob = strings.Replace(glob, `.`, `\.`, -1)
	glob = strings.Replace(glob, `*`, `.`, -1)
	glob += `$`
	regex, err = regexp.Compile(glob)
	if err != nil {
		return regex, log.Error(ctx, 400, err, `Failed to compile glob pattern on AWS input`)
	}
	return regex, nil
}

func findBibleIdMediaId(prefix string) (string, string) {
	var bibleId string
	var mediaId string
	parts := strings.Split(prefix, `/`)
	pos := len(parts) - 1
	for {
		if parts[pos] != `` {
			mediaId = parts[pos]
			bibleId = parts[pos-1]
			break
		}
		pos--
	}
	return bibleId, mediaId
}
