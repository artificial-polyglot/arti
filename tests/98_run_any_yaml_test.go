package tests

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/artificial-polyglot/arti/logger"
)

func TestRunAnyYaml(t *testing.T) {
	yamlPath := filepath.Join(os.Getenv("HOME"), "arti2", "N1SKNSEC.yaml")
	bytes, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	log.SetOutput("stderr")
	DirectSqlTest(string(bytes), []SqliteTest{}, t)
}
