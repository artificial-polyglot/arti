package s3_datastore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestS3Client(t *testing.T) {
	const BUCKET = "arti-output"

	ctx := context.Background()
	s3, err := NewS3Client(ctx)
	if err != nil {
		t.Error(err)
	}
	err = s3.PutString(BUCKET, "GaryNTest1/TestStringKey", "TestStringObject")
	if err != nil {
		t.Error(err)
	}
	filePath := filepath.Join(os.Getenv("GOPROJ"), "utility", "s3_datastore", "s3_datastore.go")
	err1 := s3.PutFile(BUCKET, "GaryNTest1", filePath)
	if err1 != nil {
		t.Error(err1)
	}
	objects, err2 := s3.ListObjects(BUCKET, "GaryNTest1")
	if err2 != nil {
		t.Error(err2)
	}
	for _, p := range objects {
		fmt.Println("object", p)
	}
	prefixes, err3 := s3.ListPrefixes(BUCKET, "EmW/")
	if err3 != nil {
		t.Error(err3)
	}
	for _, p := range prefixes {
		fmt.Println("prefix", p)
	}
	bytes, err4 := s3.GetObject(BUCKET, "GaryNTest1/TestStringKey")
	if err4 != nil {
		t.Error(err4)
	}
	fmt.Println("GetObject", string(bytes))
}

func TestPutDirectory(t *testing.T) {
	const BUCKET = "arti-output"
	ctx := context.Background()
	s3, err := NewS3Client(ctx)
	if err != nil {
		t.Error(err)
	}
	err = s3.PutDirectory(BUCKET, "LEICA", "/Users/gary/Desktop/Leica")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOutput(t *testing.T) {
	const BUCKET = "arti-output"
	ctx := context.Background()
	s3, err := NewS3Client(ctx)
	if err != nil {
		t.Error(err)
	}
	bytes, err1 := s3.GetObject(BUCKET, "GaryNTest/N2QAEBSP/00038/output/N2QAEBSP.csv")
	if err1 != nil {
		t.Error(err1)
	}
	pathFile := filepath.Join(os.Getenv("HOME"), "Downloads", "N2QAEBSP_00038.csv")
	err2 := os.WriteFile(pathFile, bytes, 0644)
	if err2 != nil {
		t.Error(err2)
	}
}
