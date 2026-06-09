package script_reader

import (
	"context"
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/performance"
)

type testFile struct {
	name string
	xlsx string
	xls  string
}

func TestScriptReader(t *testing.T) {
	log.SetOutput("stderr")
	timer := performance.NewCodeTimer()
	var files []testFile
	var f testFile
	f.name = "TGK"
	f.xlsx = "./test_data/CORE_Scr_1065p_1Eng__14_Spkr_Tajik_N2_TGK_IBT_arti.xlsx"
	f.xls = "./test_data/CORE_Scr_1065p_1Eng__14_Spkr_Tajik_N2_TGK_IBT.xls"
	files = append(files, f)
	f.name = "PKR"
	f.xlsx = "./test_data/CORE_Scr_E_2110h_Kurumba Pal_ N2PKRNLC Day 057 - Copy_arti.xlsx"
	f.xls = "./test_data/CORE_Scr_E_2110h_Kurumba Pal_ N2PKRNLC Day 057 - Copy.xls"
	files = append(files, f)
	for _, f := range files {
		var col request.SheetColumns
		processTest(f.xlsx, "./test_data/"+f.name+"_xlsx.db", col, t)
		timer.Duration("after xlsx")
		col.BookCol = 1
		col.ChapterCol = 2
		col.VerseCol = 3
		col.ActorCol = 7
		col.CharacterCol = 4
		col.LineCol = 8
		col.TextCol = 12
		processTest(f.xls, "./test_data/"+f.name+"_xls.db", col, t)
		timer.Duration("after xls")
	}
}

func DumpRows(rows [][]string) {
	for i, r := range rows {
		if i < 100 {
			fmt.Println(i, r)
		}
	}
}

func processTest(filename string, database string, col request.SheetColumns, t *testing.T) {
	db.DestroyDatabase(database)
	log.Info(context.Background(), "Start", database)
	conn := db.NewDBAdapter(context.Background(), database)
	testament := request.Testament{OT: true, NT: true}
	script := NewScriptReader(conn, testament, col)
	status := script.Read(filename)
	if status != nil {
		t.Error(status)
	}
	conn.Close()
}
