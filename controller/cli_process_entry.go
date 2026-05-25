package controller

import (
	"context"
	log "github.com/artificial-polyglot/arti/logger"
	"io"
	"os"
	"path/filepath"
)

func CLIProcessEntry(yaml []byte) (OutputFiles, *log.Status) {
	var output OutputFiles
	var status *log.Status
	var ctx = context.WithValue(context.Background(), `runType`, `cli`)
	var control = NewController(ctx, yaml)
	output, status = control.ProcessV2()
	if status != nil {
		return output, status
	}
	var err error
	for i, file := range output.FilePaths {
		ext := filepath.Ext(file)
		filename := filepath.Base(file)
		var targetPath string
		if ext == ".db" {
			filename = filename[:len(filename)-len(ext)] + ".sqlite"
			targetPath = filepath.Join(output.Directory, filename)
			err = copyFile(file, targetPath)
		} else {
			targetPath = filepath.Join(output.Directory, filename)
			err = os.Rename(file, targetPath)
		}
		if err == nil {
			output.FilePaths[i] = targetPath
		}
	}
	return output, status
}

func copyFile(src, dst string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	return err
}
