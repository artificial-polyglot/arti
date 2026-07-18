package s3_datastore

import (
	"context"
	"fmt"
	"testing"
)

func TestSignURL(t *testing.T) {
	s3, status := NewS3Client(context.Background())
	if status != nil {
		t.Fatal(status)
	}
	signed := s3.SignAudioURL("s3://arti-input/N2QAEBSP/N2QAEBSP Chapter VOX/", "N2_QAE_BSP_001_MAT_001_VOX.mp3")
	fmt.Println(signed)
}
