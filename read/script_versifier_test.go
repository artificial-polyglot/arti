package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/decode_yaml/request"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

func TestScriptVersifier(t *testing.T) {
	log.SetOutput("stderr")
	ctx := context.Background()
	conn, status := db.NewerDBAdapter(ctx, true, request.GetTestUser(), "script_versifier_test")
	if status != nil {
		t.Fatal(status)
	}
	filename := filepath.Join(os.Getenv(`HOME`), "Downloads", "N2IKHMLT Aokho (IKH)", "Text_Aokho_N2IKHMLT.xlsx")
	fmt.Println(`Filename:`, filename)
	testament := request.Testament{OT: true, NT: true}
	reader := NewScriptReader(conn, testament)
	status = reader.Read(filename)
	if status != nil {
		t.Fatal(status)
	}
	versifier := NewScriptVersifier(conn)
	newConn, status := versifier.Process()
	if status != nil {
		t.Fatal(status)
	}
	err := dumpScript(newConn, "xlsx.txt")
	if err != nil {
		t.Fatal(err)
	}
}

// Now create an equivalent database from USX data
func TestEquivalentUSXFiles(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(os.Getenv(`HOME`), "Downloads", "N2IKHMLT Aokho (IKH)", "N2IKHMLT Text", "USX")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var files []input.InputFile
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "usx") {
			var file input.InputFile
			file.Directory = directory
			file.Filename = entry.Name()
			file.BookId = file.Filename[2:5]
			files = append(files, file)
		}
	}
	conn, status := db.NewerDBAdapter(ctx, true, request.GetTestUser(), "script_versifier_usx")
	if status != nil {
		t.Fatal(status)
	}
	parser := NewUSXParser(conn)
	status = parser.ProcessFiles(files)
	if status != nil {
		t.Fatal(status)
	}
	err = dumpScript(conn, "usx.txt")
	if err != nil {
		t.Fatal(err)
	}
}

func dumpScript(conn db.DBAdapter, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	rows, err := conn.DB.Query("SELECT book_id, chapter_num, verse_str, script_text FROM scripts")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var bookId string
		var chapterNum int
		var verseStr string
		var scriptText string
		err = rows.Scan(&bookId, &chapterNum, &verseStr, &scriptText)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(file, "%s %d:%s %s|\n", bookId, chapterNum, verseStr, scriptText)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}
