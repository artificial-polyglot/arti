package mms_align

import (
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	var output []db.Output
	ctx := database.Ctx
	audioFiles, status := db.SelectAudioFiles(database)
	if status != nil {
		return output, status
	}
	var ts MMSAlign
	ts = NewMMSAlign(ctx, database, req.LanguageISO, req.AltLanguage)
	status = ts.ProcessFiles(audioFiles)
	return output, status
}
