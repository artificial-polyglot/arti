package input

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

type InputFile struct {
	MediaId    string
	MediaType  request.MediaType
	Testament  string
	BookId     string // not used for text_plain
	BookSeq    string
	Chapter    int // only used for audio
	ChapterEnd int
	Verse      string // not sure how this is used but parseV4AudioFilename parses it.
	VerseEnd   string
	ScriptLine string
	Filename   string
	FileExt    string
	Directory  string
	BaseURL    string
}

func (i *InputFile) FilePath() string {
	return filepath.Join(i.Directory, i.Filename)
}

func (i *InputFile) Download(ctx context.Context, client s3_datastore.S3Client, tempDir string) *log.Status {
	var status *log.Status
	if strings.HasPrefix(i.BaseURL, "file://") {
		i.Directory = i.BaseURL[7:]
		return nil
	}
	if strings.HasPrefix(i.BaseURL, "s3://") {
		s3Path := i.BaseURL[5:]
		slash := strings.Index(s3Path, "/")
		bucket := s3Path[0:slash]
		prefix := s3Path[slash+1:]
		var objectKey string
		if strings.HasSuffix(prefix, "/") {
			objectKey = prefix + i.Filename
		} else {
			objectKey = prefix + "/" + i.Filename
		}
		i.Directory = tempDir
		status = client.DownloadFile(bucket, objectKey, i.FilePath())
		return status
	} else {
		return log.ErrorNoErr(ctx, 500, "InputFile.Download supports s3://, and file://")
	}
}

func InsertAudioFiles(db db.DBAdapter, files []InputFile) *log.Status {
	query := `INSERT INTO audio_files (
        media_id, media_type, testament, book_id, book_seq, chapter,
        script_line, filename, file_ext, base_url)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return log.Error(db.Ctx, 500, err, "failed to prepare InsertAudioFiles statement")
	}
	defer stmt.Close()
	for _, f := range files {
		_, err = stmt.Exec(
			f.MediaId,
			string(f.MediaType),
			f.Testament,
			f.BookId,
			f.BookSeq,
			f.Chapter,
			f.ScriptLine,
			f.Filename,
			f.FileExt,
			f.BaseURL,
		)
		if err != nil {
			return log.Error(db.Ctx, 500, err, "failed to insert audio file: "+f.Filename)
		}
	}
	return nil
}

func SelectAudioFiles(db db.DBAdapter) ([]InputFile, *log.Status) {
	query := `SELECT media_id, media_type, testament, book_id, book_seq, chapter,
        script_line, filename, file_ext, base_url
        FROM audio_files
        ORDER BY file_id`
	rows, err := db.DB.QueryContext(db.Ctx, query)
	if err != nil {
		return nil, log.Error(db.Ctx, 500, err, "failed to query audio_files")
	}
	defer rows.Close()
	var files []InputFile
	for rows.Next() {
		var f InputFile
		var mediaType string
		err = rows.Scan(
			&mediaType,
			&f.MediaId,
			&f.Testament,
			&f.BookId,
			&f.BookSeq,
			&f.Chapter,
			&f.ScriptLine,
			&f.Filename,
			&f.FileExt,
			&f.BaseURL,
		)
		if err != nil {
			return nil, log.Error(db.Ctx, 500, err, "failed to scan audio_files row")
		}
		f.MediaType = request.MediaType(mediaType)
		files = append(files, f)
	}
	if err = rows.Err(); err != nil {
		return nil, log.Error(db.Ctx, 500, err, "error iterating audio_files rows")
	}
	return files, nil
}

func SelectBaseURL(db db.DBAdapter) (string, *log.Status) {
	var url string
	query := `SELECT distinct base_url FROM audio_files`
	rows, err := db.DB.QueryContext(db.Ctx, query)
	if err != nil {
		return url, log.Error(db.Ctx, 500, err, "failed to query distinct base_url")
	}
	defer rows.Close()
	var urls []string
	for rows.Next() {
		err = rows.Scan(&url)
		if err != nil {
			return url, log.Error(db.Ctx, 500, err, "failed to scan baseURL")
		}
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err = rows.Err(); err != nil {
		return url, log.Error(db.Ctx, 500, err, "error iterating baseURL rows")
	}
	if len(urls) > 1 {
		return url, log.ErrorNoErr(db.Ctx, 500, "Multiple file URLs", strings.Join(urls, ","))
	}
	return url, nil
}
