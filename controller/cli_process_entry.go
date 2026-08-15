package controller

import (
	"context"
	log "github.com/artificial-polyglot/arti/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	if output.Directory != `` {
		output.Directory = expandHome(ctx, output.Directory)
		err := os.MkdirAll(output.Directory, os.ModePerm)
		if err != nil {
			log.Warn(ctx, "Failed to create Output.Directory", output.Directory, err)
		}
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
		} else {
			log.Warn(ctx, "Failed to move output file", file, "to", targetPath, err)
		}
	}
	return output, status
}

// expandHome resolves a leading "~" or "~/..." in directory to the current
// user's home directory, so an Output.Directory setting is portable across users.
func expandHome(ctx context.Context, directory string) string {
	if directory != `~` && !strings.HasPrefix(directory, `~/`) {
		return directory
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Warn(ctx, "Failed to resolve home directory for Output.Directory", directory, err)
		return directory
	}
	return filepath.Join(home, directory[1:])
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
