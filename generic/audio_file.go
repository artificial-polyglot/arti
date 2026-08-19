package generic

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
)

type AudioFile struct {
	BookId      string `json:"book_id"`
	Chapter     int    `json:"chapter"`
	Bucket      string `json:"bucket"`
	ObjectKey   string `json:"object_key"`
	UnsignedURL string `json:"unsigned_url"`
	SignedURL   string `json:"signed_url"`
}

func NewAudioFile(bookId string, chapter int, baseURL string, filename string) AudioFile {
	var file AudioFile
	file.BookId = bookId
	file.Chapter = chapter
	if strings.HasPrefix(baseURL, "file://") {
		baseURL = baseURL[7:]
		file.Bucket = ""
		file.ObjectKey = filepath.Join(baseURL, filename)
	} else if strings.HasPrefix(baseURL, "s3://") {
		baseURL = baseURL[5:]
		slash := strings.Index(baseURL, "/")
		file.Bucket = baseURL[0:slash]
		prefix := baseURL[slash+1:]
		file.ObjectKey = filepath.Join(prefix, filename)
	}
	bucket := file.internalBucketName(file.Bucket)
	file.UnsignedURL = "/file?bucket=" + bucket + "&key=" + file.ObjectKey + "&mode=play"
	return file
}

func (f *AudioFile) internalBucketName(bucket string) string {
	strings.Cut(bucket, "-")
	dash := strings.Index(bucket, "-")
	if dash < 0 {
		return bucket
	} else {
		return bucket[dash+1:]
	}
}

func (f *AudioFile) ToJSON(ctx context.Context) (string, *log.Status) {
	bytes, err := json.Marshal(f)
	if err != nil {
		return "", log.Error(ctx, 500, err, "Could not Marshal AudioFile object")
	}
	return string(bytes), nil
}
