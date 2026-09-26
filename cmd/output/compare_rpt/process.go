package compare_rpt

import (
	"encoding/json"
	"os"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	var output []db.Output

	records, fileMap, status := CompareReport(database)
	if status != nil {
		return output, status
	}

	jsonBytes, err := json.MarshalIndent(records, "", " ")
	if err != nil {
		return output, log.Error(database.Ctx, 500, err, "Error writing Pairs to json bytes")
	}
	jsonFile := ""
	err = os.WriteFile(jsonFile, jsonBytes, 0644)
	if err != nil {
		return output, log.Error(database.Ctx, 500, err, "Error writing pairs json to file")
	}
	out := db.Output{Component: "compare_rpt", Report: "", FilePath: jsonFile}
	output = append(output, out)

	jsonPath, status1 := generic.OutputAudioFiles(database.Ctx, fileMap)
	if status1 != nil {
		return output, status1
	}
	out = db.Output{Component: "compare_rpt", Report: "audio_files.json", FilePath: jsonPath}
	output = append(output, out)

	report := NewHTMLWriter(database.Ctx, "datasetname")
	filePath, status1 := report.WriteReport("baseDataset", records, req.LanguageISO, fileMap)
	if status1 != nil {
		return output, status
	}
	out = db.Output{Component: "compare_rpt", Report: "compare.html", FilePath: filePath}
	output = append(output, out)

	status = database.InsertOutput(output)
	if status != nil {
		return output, status
	}
	return output, nil
}
