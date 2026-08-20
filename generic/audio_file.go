package generic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
	if strings.HasPrefix(baseURL, "file://") { // This will not work on cloudflare.
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

func OutputAudioFiles(ctx context.Context, files map[string]AudioFile) (string, *log.Status) {
	var slice = make([]AudioFile, 0, len(files))
	for _, val := range files {
		slice = append(slice, val)
	}
	sort.Slice(slice, func(i, j int) bool {
		if slice[i].BookId != slice[j].BookId {
			return slice[i].BookId < slice[j].BookId
		}
		return slice[i].Chapter < slice[j].Chapter
	})
	bytes, err := json.Marshal(slice)
	if err != nil {
		return "", log.Error(ctx, 500, err, "Could not Marshal AudioFile object")
	}
	dir, err := os.MkdirTemp("", "output_audio_files")
	if err != nil {
		return "", log.Error(ctx, 500, err, "Could not create temp directory in OutputAudioFiles")
	}
	filePath := filepath.Join(dir, "audio_file_urls.json")
	err = os.WriteFile(filePath, bytes, 0644)
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not write to file in OutputAudioFiles")
	}
	return filePath, nil
}
