package stdio_exec

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func ExecComponentInterim(ctx context.Context, componentPath string, database db.DBAdapter, args ...string) (db.DBAdapter, *log.Status) {
	dbPath := database.DatabasePath
	database.Close()
	newDBPath, status := ExecComponent(ctx, componentPath, dbPath, args...)
	if status != nil {
		return db.DBAdapter{}, status
	}
	newDBPath = strings.TrimRight(newDBPath, "\r\n")
	newDatabase := db.NewDBAdapter(ctx, newDBPath)
	return newDatabase, nil
}

func ExecComponent(ctx context.Context, componentPath string, databasePath string, args ...string) (string, *log.Status) {
	cmd := exec.Command(componentPath, args...)
	cmd.Stdin = strings.NewReader(databasePath + "\n")
	var out, outErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	err := cmd.Run()
	if err != nil {
		return "", log.Error(ctx, 500, err, outErr.String())
	}
	return out.String(), nil
}
