package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGetAudioDuration(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), "ENGWEB", "ENGWEBN2DA-mp3-64")
	filename := "B04___09_John________ENGWEBN2DA.mp3"
	result, status := GetAudioDuration(ctx, directory, filename)
	if status != nil {
		t.Fatal(status)
	}
	if result != 363.493878 {
		t.Error("Result should be 363.493878", "result was:", result)
	}
}

func TestGetAudioSize(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), "ENGWEB", "ENGWEBN2DA-mp3-64")
	filename := "B04___09_John________ENGWEBN2DA.mp3"
	result, status := GetAudioSize(ctx, directory, filename)
	if status != nil {
		t.Fatal(status)
	}
	if result != 2908464 {
		t.Error("Result should be 363.493878")
	}
}

func TestGetAudioBitrate(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(os.Getenv("FCBH_DATASET_FILES"), "ENGWEB", "ENGWEBN2DA-mp3-64")
	filename := "B04___09_John________ENGWEBN2DA.mp3"
	result, status := GetAudioBitrate(ctx, directory, filename)
	if status != nil {
		t.Fatal(status)
	}
	if result != 64011 {
		t.Error("Result should be 64011")
	}
}
