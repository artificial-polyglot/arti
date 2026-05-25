package output

import (
	"encoding/csv"
	"encoding/json"
	log "github.com/artificial-polyglot/arti/logger"
	"os"
	"path/filepath"
	"strconv"
)

func (o *Output) JSONStatus(status log.Status, debug bool) (string, *log.Status) {
	var filename string
	file, err := os.Create(filepath.Join(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+".json"))
	//file, err := os.CreateTemp(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+"_*.json")
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, status.Err)
	}
	filename = file.Name()
	if !debug {
		status.Trace = ``
	}
	bytes, err := json.MarshalIndent(status, "", "   ")
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, status.Err)
	}
	_, _ = file.Write(bytes)
	_ = file.Close()
	return filename, nil
}

func (o *Output) CSVStatus(status log.Status, debug bool) (string, *log.Status) {
	var filename string
	file, err := os.Create(filepath.Join(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+".csv"))
	//file, err := os.CreateTemp(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+"_*.csv")
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, status.Err)
	}
	filename = file.Name()
	writer := csv.NewWriter(file)
	_ = writer.Write([]string{`Name`, `Value`})
	_ = writer.Write([]string{`status`, strconv.Itoa(status.Status)})
	_ = writer.Write([]string{`message`, status.Message})
	_ = writer.Write([]string{`error`, status.Err})
	_ = writer.Write([]string{`request`, status.Request})
	if debug {
		_ = writer.Write([]string{`trace`, status.Trace})
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, status.Err)
	}
	_ = file.Close()
	return filename, nil
}
