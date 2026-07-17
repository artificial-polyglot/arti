package qa_align

import (
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func Process(database db.DBAdapter) ([]db.Output, *log.Status) {
	var output []db.Output
	req, status := database.SelectRequest()
	if status != nil {
		return output, status
	}
	asr := NewQAAlign(database.Ctx, database, req.LanguageISO, req.AltLanguage, true)
	status = asr.ProcessFiles()
	if status != nil {
		return output, status
	}
	return output, nil
}
