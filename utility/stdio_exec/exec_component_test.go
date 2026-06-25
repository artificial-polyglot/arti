package stdio_exec

import (
	"context"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"
)

func TestExecComponent(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	user := request.GetTestUser()
	conn, status := db.NewerDBAdapter(ctx, false, user, "PlainTextEditScript_ENGWEB")
	if status != nil {
		t.Fatal(status)
	}
	count1, status1 := conn.CountScriptRows()
	if status1 != nil {
		t.Fatal(status1)
	}
	// This test calls shell command echo and passes it the databasePath
	newDB, status2 := ExecComponentInterim(ctx, "echo", conn, conn.DatabasePath)
	if status2 != nil {
		t.Fatal(status2)
	}
	count2, status3 := newDB.CountScriptRows()
	if status3 != nil {
		t.Fatal(status3)
	}
	if count1 != count2 {
		t.Error("The database count should not have changed, orig:", count1, "after:", count2)
	}
}
