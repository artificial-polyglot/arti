package tests

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/artificial-polyglot/arti/logger"
)

func TestRunAnyYaml(t *testing.T) {
	yamlPath := filepath.Join(os.Getenv("HOME"), "arti2", "N1SKNSEC_rpt.yaml")
	bytes, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(bytes)
	log.SetOutput("stderr")
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
