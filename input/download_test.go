package input

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadToDBLocation_S3(t *testing.T) {
	ctx := context.Background()
	username := "Tests"
	project := "17a_mms_adapter"
	localDBPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), username, project+".db")
	err := os.RemoveAll(localDBPath)
	if err != nil {
		t.Fatal(err)
	}
	s3Name := "s3://arti-input/17a_mms_adapter_test_data/" + project + ".db"
	resultName, status := DownloadToDBLocation(ctx, s3Name, username)
	if status != nil {
		t.Error(status)
	}
	if resultName != project {
		t.Error("Result path is not correct")
	}
	info, err := os.Stat(localDBPath)
	if err != nil {
		t.Error(err)
	}
	if info.Size() != 1396736 {
		t.Error("File is expected to be 1396736, but is", info.Size())
	}
}
