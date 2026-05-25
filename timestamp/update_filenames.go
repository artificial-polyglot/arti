package timestamp

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/ffmpeg"
)

type audioFile struct {
	scriptLine string
	filename   string
	beginTS    float64
	endTS      float64
}

func UpdateFilenames(ctx context.Context, conn db.DBAdapter, files []input.InputFile) *log.Status {
	var status *log.Status
	var results []audioFile
	for _, file := range files {
		var ts audioFile
		ts.scriptLine = file.ScriptLine
		ts.filename = file.Filename
		ts.beginTS = 0.0
		ts.endTS, status = ffmpeg.GetAudioDuration(ctx, file.Directory, file.Filename)
		if status != nil {
			return status
		}
		results = append(results, ts)
	}
	err := updateScriptTimestamps(conn, results)
	if err != nil {
		return log.Error(ctx, 500, err, "Error in timestamp.updateFilenames")
	}
	return nil
}

func updateScriptTimestamps(conn db.DBAdapter, files []audioFile) error {
	query := `UPDATE scripts SET audio_file = ?, script_begin_ts = ?,
		script_end_ts = ? WHERE script_num = ?`
	tx, err := conn.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, rec := range files {
		_, err = stmt.Exec(rec.filename, rec.beginTS, rec.endTS, rec.scriptLine)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	return err
}
