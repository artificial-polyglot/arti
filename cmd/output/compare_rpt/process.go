package compare_rpt

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	var output []db.Output
	ctx := database.Ctx

	records, fileMap, status := CompareReport(database)
	if status != nil {
		return output, status
	}
	s3, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		log.Warn(ctx, "Could not obtain s3 client to sign URL")
	} else {
		s3.SignAudioFiles(fileMap)
	}

	jsonBytes, err := json.MarshalIndent(records, "", " ")
	if err != nil {
		return output, log.Error(database.Ctx, 500, err, "Error writing Pairs to json bytes")
	}
	jsonFile := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), req.Username, req.DatasetName+"_pairs.json")
	err = os.WriteFile(jsonFile, jsonBytes, 0644)
	if err != nil {
		return output, log.Error(database.Ctx, 500, err, "Error writing pairs json to file")
	}
	out := db.Output{Component: "compare_rpt", Report: "pairs.json", FilePath: jsonFile}
	output = append(output, out)

	jsonPath, status1 := generic.OutputAudioFiles(database.Ctx, fileMap)
	if status1 != nil {
		return output, status1
	}
	out = db.Output{Component: "compare_rpt", Report: "audio_files.json", FilePath: jsonPath}
	output = append(output, out)

	report := NewHTMLWriter(database.Ctx, req.DatasetName)
	filePath, status1 := report.WriteReport(records, req.LanguageISO, fileMap)
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
