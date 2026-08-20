package proofing_rpt

import (
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
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
	records, verses, audioURLs, status := rpt.Process()
	if status != nil {
		return output, status
	}
	jsonName, status1 := generic.OutputAudioFiles(database.Ctx, audioURLs)
	if status1 != nil {
		return output, status
	}
	out := db.Output{Component: "proofing_rpt", Report: "audio_urls", FilePath: jsonName}
	output = append(output, out)
	writer := NewHTMLWriter(database.Ctx, req.DatasetName)
	filename, status := writer.WriteReport(records, verses, audioURLs, req.LanguageISO, req.SpeechToText)
	if status != nil {
		return output, status
	}
	out = db.Output{Component: "proofing_rpt", Report: "proofing", FilePath: filename}
	output = append(output, out)
	status = database.InsertOutput(output)
	if status != nil {
		return output, status
	}
	return output, nil
}
