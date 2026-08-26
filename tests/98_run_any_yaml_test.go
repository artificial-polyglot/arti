package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

func TestRunAnyYaml(t *testing.T) {
	courier.IsCourierTest = true
	yamlPath := filepath.Join(os.Getenv("HOME"), "arti2", "N1SKNSEC_rpt.yaml")
	bytes, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(bytes)
	//log.SetOutput("stderr")
	DirectSqlTest(yaml, []SqliteTest{}, t)
}
