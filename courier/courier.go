package courier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type model struct {
	modelType   string
	languageISO string
}

type Courier struct {
	ctx         context.Context
	start       time.Time
	bucket      string
	Component   string
	username    string
	dataset     string
	run         int
	yamlContent string
	logFile     string
	databases   []string
	outputs     []string
	outputKeys  []string
	models      []model
}

var IsCourierTest = false

func NewCourier(ctx context.Context, yaml []byte) Courier {
	var b Courier
	b.ctx = ctx
	b.start = time.Now()
	b.bucket = os.Getenv("FCBH_DATASET_IO_BUCKET")
	b.Component = "arti" // default
	b.yamlContent = string(yaml)
	b.username = ParseYaml(b.yamlContent, `username`)
	b.dataset = ParseYaml(b.yamlContent, `dataset_name`)

	logFile := os.Getenv("FCBH_DATASET_LOG_FILE")
	b.AddLogFile(logFile)
	return b
}

func (b *Courier) AddLogFile(logPath string) {
	b.logFile = logPath
	if !IsCourierTest {
		err := os.Truncate(logPath, 0)
		if err != nil {
			log.Warn(b.ctx, "Failed to truncate log file", err)
		}
	}
}

func (b *Courier) AddDatabase(conn db.DBAdapter) {
	b.databases = append(b.databases, conn.DatabasePath)
}

func (b *Courier) AddOutput(outputPath string) {
	if len(outputPath) > 0 {
		b.outputs = append(b.outputs, outputPath)
	}
}

func (b *Courier) AddJson(records any, filePath string) {
	jsonData, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		log.Warn(b.ctx, err, "Failed to marshal ", filePath)
	} else {
		err = os.WriteFile(filePath, jsonData, 0644)
		if err != nil {
			log.Warn(b.ctx, err, "Failed to write ", filePath)
		} else {
			b.AddOutput(filePath)
		}
	}
}

func (b *Courier) AddModel(modelType string, languageISO string) {
	b.models = append(b.models, model{modelType: modelType, languageISO: languageISO})
}

func (b *Courier) GetOutputPaths() []string {
	return b.outputs
}

func (b *Courier) GetOutputByExt() []string {
	var results []string
	for _, path := range b.outputs {
		if !strings.HasSuffix(path, ".db") && !strings.HasSuffix(path, ".sqlite") {
			results = append(results, path)
		}
	}
	return results
}

func (b *Courier) PersistToBucket(runStatus *log.Status) *log.Status {
	var allStatus []*log.Status
	if !testing.Testing() || IsCourierTest {
		client, status := s3_datastore.NewS3Client(b.ctx)
		if status != nil {
			return status
		}
		var run int
		run, status = b.findLastRun(client)
		allStatus = append(allStatus, status)
		run++
		_, status = b.uploadString(client, run, "request", b.dataset+".yaml", b.yamlContent)
		allStatus = append(allStatus, status)
		_, status = b.uploadFile(client, run, "log", b.logFile, "text/plain", true)
		allStatus = append(allStatus, status)
		for _, database := range b.databases {
			_, status = b.uploadFile(client, run, "database", database, "application/x-sqlite3", false)
			allStatus = append(allStatus, status)
		}
		for _, output := range b.outputs {
			var mimeType string
			var inline bool
			switch strings.ToLower(filepath.Ext(output)) {
			case ".html":
				mimeType = "text/html"
				inline = true
			case ".csv":
				mimeType = "text/csv"
				inline = true
			case ".json":
				mimeType = "text/json"
				inline = true
			default:
				mimeType = ""
				inline = false
			}
			outputKey, status2 := b.uploadFile(client, run, "output", output, mimeType, inline)
			allStatus = append(allStatus, status2)
			b.outputKeys = append(b.outputKeys, outputKey)
		}
		if runStatus == nil {
			bucket := os.Getenv("FCBH_MODELS_BUCKET")
			for _, modl := range b.models {
				prefix := filepath.Join(modl.modelType, modl.languageISO)
				localDir := filepath.Join(os.Getenv("FCBH_DATASET_DB"), prefix)
				lastModelRun, status3 := b.findLastModelRun(client, bucket, prefix)
				allStatus = append(allStatus, status3)
				modelRunStr := fmt.Sprintf("%05d", lastModelRun+1)
				remotePrefix := filepath.Join(prefix, modelRunStr)
				status3 = client.PutDirectory(bucket, remotePrefix, localDir)
				allStatus = append(allStatus, status3)
			}
		}
		if runStatus != nil {
			_, status = b.uploadString(client, run, "status", "Error", runStatus.String())
		}
		loc, _ := time.LoadLocation("America/Denver")
		_, status = b.uploadString(client, run, "runtime", b.start.In(loc).Format(`Mon Jan 2 2006 03:04:05 pm MST`), "")
		allStatus = append(allStatus, status)
		_, status = b.uploadString(client, run, "duration", time.Since(b.start).String(), "")
		allStatus = append(allStatus, status)
		for _, stat := range allStatus {
			if stat != nil {
				status = stat
				break
			}
		}
	}
	return nil
}

func ParseYaml(yamlContent string, name string) string {
	var result string
	index := strings.Index(yamlContent, name+":")
	if index == -1 {
		result = "unknown-" + name
	} else {
		start := index + len(name) + 1
		end := strings.Index(yamlContent[start:], "\n")
		result = strings.TrimSpace(yamlContent[start : start+end])
	}
	return result
}

func (b *Courier) findLastRun(client s3_datastore.S3Client) (int, *log.Status) {
	if testing.Testing() {
		prefix := b.username + "/" + b.dataset + "/"
		status := client.ClearDirectory(b.bucket, prefix)
		return 0, status
	}
	var result int
	var status *log.Status
	prefix := b.username + "/" + b.dataset + "/" + b.Component + "/"
	output, err := client.Client.ListObjectsV2(b.ctx, &s3.ListObjectsV2Input{
		Bucket: &b.bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return result, log.Error(b.ctx, 500, err, "Error listing bucket objects.")
	}
	maxRun := 0
	for _, obj := range output.Contents {
		parts := strings.Split(*obj.Key, "/")
		if len(parts) < 5 {
			continue
		}
		runStr := parts[3]
		var runNum int
		runNum, err = strconv.Atoi(runStr)
		if err != nil {
			return result, log.Error(b.ctx, 500, err, "Error converting run number to int; value:", runStr)
		}
		if runNum > maxRun {
			maxRun = runNum
		}
	}
	return maxRun, status
}

// findLastModelRun returns the highest existing run number (the 5-digit
// zero-filled directory - e.g. mms_adapters/atg/00001/ - one level below
// prefix) so the caller can upload to lastRun+1. Zero, with no error, means
// no run directories exist yet (first upload for this model/language).
func (b *Courier) findLastModelRun(client s3_datastore.S3Client, bucket string, prefix string) (int, *log.Status) {
	var result int
	prefixes, status := client.ListPrefixes(bucket, prefix+"/")
	if status != nil {
		return result, status
	}
	maxRun := 0
	for _, p := range prefixes {
		parts := strings.Split(strings.TrimSuffix(p, "/"), "/")
		runStr := parts[len(parts)-1]
		runNum, err := strconv.Atoi(runStr)
		if err != nil {
			return result, log.Error(b.ctx, 500, err, "Error converting model run number to int; value:", runStr)
		}
		if runNum > maxRun {
			maxRun = runNum
		}
	}
	return maxRun, status
}

func (b *Courier) uploadString(client s3_datastore.S3Client, run int, typ string, filename string, content string) (string, *log.Status) {
	var objectKey string
	var status *log.Status
	objectKey = b.createKey(run, typ, filename)
	status = client.PutString(b.bucket, objectKey, content)
	if status != nil {
		return objectKey, status
	}
	return objectKey, status
}

func (b *Courier) uploadFile(client s3_datastore.S3Client, run int, typ string, filePath string,
	mimeType string, inline bool) (string, *log.Status) {
	var objectKey string
	var status *log.Status
	objectKey = b.createKey(run, typ, filePath)
	status = client.PutFile(b.bucket, objectKey, filePath, mimeType, inline)
	if status != nil {
		return objectKey, status
	}
	return objectKey, status
}

func (b *Courier) createKey(run int, typ string, filename string) string {
	runStr := fmt.Sprintf("%05d", run)
	filename = filepath.Base(filename)
	return strings.Join([]string{b.username, b.dataset, b.Component, runStr, typ, filename}, "/")
}
