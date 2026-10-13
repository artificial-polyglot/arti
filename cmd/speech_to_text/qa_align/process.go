package qa_align

import (
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	var output []db.Output
	var modelBase string
	if !req.Training2.NoTraining {
		modelBase = req.Training2.ModelBase
	} else if !req.Training.NoTraining {
		modelBase = "mms_adapters"
	} else {
		modelBase = "mms"
	}
	asr := NewQAAlign(database.Ctx, database, req.LanguageISO, req.AltLanguage, modelBase, req.Testament)
	status := asr.ProcessFiles()
	if status != nil {
		return output, status
	}
	return output, nil
}
