package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/request"
)

func TestScriptReader(t *testing.T) {
	bibleId := `ATIWBT`
	database := bibleId + "_SCRIPT.db"
	db.DestroyDatabase(database)
	conn := db.NewDBAdapter(context.Background(), database)
	testament := request.Testament{OT: true, NT: true}
	script := NewScriptReader(conn, testament)
	filename := filepath.Join(os.Getenv(`FCBH_DATASET_FILES`), bibleId, bibleId+`N2ST.xlsx`)
	fmt.Println(`Filename:`, filename)
	status := script.Read(filename)
	if status != nil {
		t.Fatal(status)
	}
	conn.Close()
}
