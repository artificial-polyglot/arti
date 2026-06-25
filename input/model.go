package input

import (
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"

	"path/filepath"
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
}

func (i *InputFile) FilePath() string {
	return filepath.Join(i.Directory, i.Filename)
}

func InsertAudioFiles(db db.DBAdapter, files []InputFile) *log.Status {
	query := `INSERT INTO audio_files (
        media_id, media_type, testament, book_id, book_seq, chapter,
        script_line, filename, file_ext, directory)
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
			f.Directory,
		)
		if err != nil {
			return log.Error(db.Ctx, 500, err, "failed to insert audio file: "+f.Filename)
		}
	}
	return nil
}

func SelectAudioFiles(db db.DBAdapter) ([]InputFile, *log.Status) {
	query := `SELECT media_id, media_type, testament, book_id, book_seq, chapter,
        script_line, filename, file_ext, directory
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
			&f.Directory,
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
