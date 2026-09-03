package qa_align

import (
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	var output []db.Output
	asr := NewQAAlign(database.Ctx, database, req.LanguageISO, req.AltLanguage, true, req.Testament)
	status := asr.ProcessFiles()
	if status != nil {
		return output, status
	}
	return output, nil
}
