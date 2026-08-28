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

	calc := NewAlignSilence(database)
	faLines, audioURLs, status := calc.Process()
	if status != nil {
		return output, status
	}

	writer := NewHTMLWriter(database.Ctx, database.Project)
	filename, status := writer.WriteReport(faLines, audioURLs, req.LanguageISO, req.SpeechToText)
	if status != nil {
		return output, status
	}
	out := db.Output{Component: "proofing_rpt", Report: "proofing", FilePath: filename}
	output = append(output, out)

	jsonName, status1 := generic.OutputAudioFiles(database.Ctx, audioURLs)
	if status1 != nil {
		return output, status
	}
	out = db.Output{Component: "proofing_rpt", Report: "audio_urls", FilePath: jsonName}
	output = append(output, out)
	status = database.InsertOutput(output)
	if status != nil {
		return output, status
	}
	return output, nil
}
