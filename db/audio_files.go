package db

import (
	"strings"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func InsertAudioFiles(conn DBAdapter, files []generic.InputFile) *log.Status {
	query := "DELETE FROM audio_files"
	_, err := conn.DB.Exec(query)
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "Failed to delete audio_files before insert")
	}
	query = `INSERT INTO audio_files (
        media_id, media_type, testament, book_id, book_seq, chapter,
        script_line, filename, file_ext, base_url)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := conn.DB.Prepare(query)
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "failed to prepare InsertAudioFiles statement")
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
			return log.Error(conn.Ctx, 500, err, "failed to insert audio file: "+f.Filename)
		}
	}
	return nil
}

func SelectAudioFiles(conn DBAdapter) ([]generic.InputFile, *log.Status) {
	query := `SELECT media_id, media_type, testament, book_id, book_seq, chapter,
        script_line, filename, file_ext, base_url
        FROM audio_files
        ORDER BY file_id`
	rows, err := conn.DB.QueryContext(conn.Ctx, query)
	if err != nil {
		return nil, log.Error(conn.Ctx, 500, err, "failed to query audio_files")
	}
	defer rows.Close()
	var files []generic.InputFile
	for rows.Next() {
		var f generic.InputFile
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
			return nil, log.Error(conn.Ctx, 500, err, "failed to scan audio_files row")
		}
		f.MediaType = request.MediaType(mediaType)
		files = append(files, f)
	}
	if err = rows.Err(); err != nil {
		return nil, log.Error(conn.Ctx, 500, err, "error iterating audio_files rows")
	}
	return files, nil
}

func SelectBaseURL(conn DBAdapter) (string, *log.Status) {
	var url string
	query := `SELECT distinct base_url FROM audio_files`
	rows, err := conn.DB.QueryContext(conn.Ctx, query)
	if err != nil {
		return url, log.Error(conn.Ctx, 500, err, "failed to query distinct base_url")
	}
	defer rows.Close()
	var urls []string
	for rows.Next() {
		err = rows.Scan(&url)
		if err != nil {
			return url, log.Error(conn.Ctx, 500, err, "failed to scan baseURL")
		}
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err = rows.Err(); err != nil {
		return url, log.Error(conn.Ctx, 500, err, "error iterating baseURL rows")
	}
	if len(urls) > 1 {
		return url, log.ErrorNoErr(conn.Ctx, 500, "Multiple file URLs", strings.Join(urls, ","))
	}
	return url, nil
}

func UpdateIdent(conn DBAdapter, ident *Ident, textFiles []generic.InputFile, audioFiles []generic.InputFile) *log.Status {
	var status *log.Status
	_ = updateIdentText(ident, textFiles)
	_ = updateIdentAudio(ident, audioFiles)
	status = conn.InsertReplaceIdent(*ident)
	return status
}

func updateIdentText(ident *Ident, files []generic.InputFile) bool {
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

func updateIdentAudio(ident *Ident, files []generic.InputFile) bool {
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
