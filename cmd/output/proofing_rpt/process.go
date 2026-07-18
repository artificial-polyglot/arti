package proofing_rpt

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

	// get uroman from request?, or make it a report option
	rpt := NewProofingRpt(database.Ctx, database, req.LanguageISO, false)
	records, verses, baseURL, status := rpt.Process()
	if status != nil {
		return output, status
	}

	writer := NewHTMLWriter(database.Ctx, req.DatasetName)
	filename, status := writer.WriteReport(records, verses, baseURL, req.LanguageISO, req.SpeechToText)
	if status != nil {
		return output, status
	}
	var out = db.Output{Component: "proofing_rpt", Report: "proofing", FilePath: filename}
	output = append(output, out)
	status = database.InsertOutput(output)
	if status != nil {
		return output, status
	}
	return output, nil
}
