package output

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"gonum.org/v1/gonum/floats"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestJSONWriterScript(t *testing.T) {
	ctx := context.Background()
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	//prepareTimestampAndFMCCData(conn, `ENGWEB`, `ENGWEBN2DA`, t)
	var out = NewOutput(ctx, conn, `TestScripts`, false, false)
	structs, meta := out.PrepareScripts()
	fmt.Println("Loaded Scripts", len(structs))
	filename, status := out.WriteJSON(structs, meta)
	if status != nil {
		t.Fatal(status)
	}
	fileRecs := readJSONScript(filename, t)
	fmt.Println(len(fileRecs))
	dbRecs, status := out.LoadScriptStruct(conn)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("Written JSON", filename)
	compareScript(dbRecs, fileRecs, t)
	fmt.Println("Written JSON", filename)
}

func TestJSONWriterWord(t *testing.T) {
	ctx := context.Background()
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	//prepareTimestampAndFMCCData(conn, `ENGWEB`, `ENGWEBN2DA`, t)
	var out = NewOutput(ctx, conn, `TestWords`, false, false)
	structs, meta := out.PrepareWords()
	fmt.Println("Loaded Scripts", len(structs))
	filename, status := out.WriteJSON(structs, meta)
	if status != nil {
		t.Fatal(status)
	}
	fileRecs := readJSONWords(filename, t)
	dbRecs, status := out.LoadWordStruct(conn)
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("Written JSON", filename)
	compareWords(dbRecs, fileRecs, t)
	fmt.Println("Written JSON", filename)
}

func readJSONScript(filename string, t *testing.T) []Script {
	var results = make([]Script, 0, 10000)
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Error(err)
	}
	var records []map[string]any
	err = json.Unmarshal(content, &records)
	if err != nil {
		t.Error(err)
	}
	var sc Script
	for _, rec := range records {
		if rec["script_id"] != nil {
			if sc.ScriptId != 0 {
				results = append(results, sc)
			}
			sc = Script{}
			sc.BookId = rec["book_id"].(string)
			sc.ChapterNum = int(rec["chapter_num"].(float64))
			sc.Reference = rec["reference"].(string)
			sc.ScriptBeginTS = rec["script_begin_ts"].(float64)
			sc.ScriptEndTS = rec["script_end_ts"].(float64)
			sc.ScriptId = int(rec["script_id"].(float64))
			sc.ScriptNum = rec["script_num"].(string)
			sc.ScriptText = rec["script_text"].(string)
			sc.VerseStr = rec["verse_str"].(string)
		}
		var mfcc = make([]float64, 300)
		var cols = 0
		for key, val := range rec {
			if strings.HasPrefix(key, "mfcc") {
				num, err := strconv.Atoi(key[4:])
				if err != nil {
					t.Error(err)
				}
				mfcc[num] = val.(float64)
				if num > cols {
					cols = num
				}
			}
		}
		sc.MFCCRows++
		sc.MFCCCols = cols + 1
		sc.MFCC = append(sc.MFCC, mfcc[:sc.MFCCCols])
	}
	results = append(results, sc)
	return results
}

func readJSONWords(filename string, t *testing.T) []Word {
	var results = make([]Word, 0, 10000)
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Error(err)
	}
	var records []map[string]any
	err = json.Unmarshal(content, &records)
	if err != nil {
		t.Error(err)
	}
	var wd Word
	for _, rec := range records {
		if rec["word_id"] != nil {
			if wd.WordId != 0 {
				results = append(results, wd)
			}
			wd = Word{}
			wd.BookId = rec["book_id"].(string)
			wd.ChapterNum = int(rec["chapter_num"].(float64))
			wd.Reference = rec["reference"].(string)
			wd.ScriptId = int(rec["script_id"].(float64))
			wd.VerseStr = rec["verse_str"].(string)
			wd.Word = rec["word"].(string)
			wd.WordId = int(rec["word_id"].(float64))
			wd.WordBeginTS = rec["word_begin_ts"].(float64)
			wd.WordEndTS = rec["word_end_ts"].(float64)
			wd.WordSeq = int(rec["word_seq"].(float64))
			var wrdEnc = make([]float64, 300)
			var cols = 0
			for key, val := range rec {
				if strings.HasPrefix(key, "word_enc") {
					num, err := strconv.Atoi(key[8:])
					if err != nil {
						t.Error(err)
					}
					wrdEnc[num] = val.(float64)
					if num > cols {
						cols = num
					}
				}
			}
			wd.WordEnc = wrdEnc[:cols+1]
		}
		var mfcc = make([]float64, 300)
		var cols = 0
		for key, val := range rec {
			if strings.HasPrefix(key, "mfcc") {
				num, err := strconv.Atoi(key[4:])
				if err != nil {
					t.Error(err)
				}
				mfcc[num] = val.(float64)
				if num > cols {
					cols = num
				}
			}
		}
		wd.MFCCRows++
		wd.MFCCCols = cols + 1
		wd.MFCC = append(wd.MFCC, mfcc[:wd.MFCCCols])
	}
	results = append(results, wd)
	return results
}

func compareScript(dbRecs []Script, fileRecs []Script, t *testing.T) {
	if len(dbRecs) != len(fileRecs) {
		t.Error(`dbRecs has length`, len(dbRecs), `fileRecs has length`, len(fileRecs))
	}
	fmt.Println(`Compare `, len(dbRecs), `records`)
	var rowDiffCount = 0
	for i, db := range dbRecs {
		file := fileRecs[i]
		if db.BookId != file.BookId {
			t.Error(`BookId mismatch`, db.BookId, file.BookId)
		}
		if db.ChapterNum != file.ChapterNum {
			t.Error(`ChapterNum mismatch`, db.ChapterNum, file.ChapterNum)
		}
		if db.Reference != file.Reference {
			t.Error(`Reference mismatch`, db.Reference, file.Reference)
		}
		if db.ScriptBeginTS != file.ScriptBeginTS {
			t.Error(`ScriptBeginTS mismatch`, db.ScriptBeginTS, file.ScriptBeginTS)
		}
		if db.ScriptEndTS != file.ScriptEndTS {
			t.Error(`ScriptEndTS mismatch`, db.ScriptEndTS, file.ScriptEndTS)
		}
		if db.ScriptId != file.ScriptId {
			t.Error(`ScriptId mismatch`, db.ScriptId, file.ScriptId)
		}
		if db.ScriptNum != file.ScriptNum {
			t.Error(`ScriptNum mismatch`, db.ScriptNum, file.ScriptNum)
		}
		if db.ScriptText != file.ScriptText {
			t.Error(`ScriptText mismatch`, db.ScriptText, file.ScriptText)
		}
		if db.VerseStr != file.VerseStr {
			t.Error(`VerseStr mismatch`, db.VerseStr, file.VerseStr)
		}
		for row := 0; row < db.MFCCRows; row++ {
			if !floats.EqualApprox(db.MFCC[row], file.MFCC[row], 0.00001) {
				t.Error(`MFCC mismatch in row`, row)
			}
		}
	}
	fmt.Println("rows", len(dbRecs), `differences`, rowDiffCount)
}

func compareWords(dbRecs []Word, fileRecs []Word, t *testing.T) {
	if len(dbRecs) != len(fileRecs) {
		t.Error(`dbRecs has length`, len(dbRecs), `fileRecs has length`, len(fileRecs))
	}
	fmt.Println(`Compare `, len(dbRecs), `records`)
	var rowDiffCount = 0
	for i, db := range dbRecs {
		file := fileRecs[i]
		if db.BookId != file.BookId {
			t.Error(`BookId mismatch`, db.BookId, file.BookId)
		}
		if db.ChapterNum != file.ChapterNum {
			t.Error(`ChapterNum mismatch`, db.ChapterNum, file.ChapterNum)
		}
		if db.Reference != file.Reference {
			t.Error(`Reference mismatch`, db.Reference, file.Reference)
		}
		if db.WordBeginTS != file.WordBeginTS {
			t.Error(`WordBeginTS mismatch`, db.WordBeginTS, file.WordBeginTS)
		}
		if db.WordEndTS != file.WordEndTS {
			t.Error(`WordEndTS mismatch`, db.WordEndTS, file.WordEndTS)
		}
		if db.ScriptId != file.ScriptId {
			t.Error(`ScriptId mismatch`, db.ScriptId, file.ScriptId)
		}
		if db.Word != file.Word {
			t.Error(`ScriptText mismatch`, db.Word, file.Word)
		}
		if db.VerseStr != file.VerseStr {
			t.Error(`VerseStr mismatch`, db.VerseStr, file.VerseStr)
		}
		if !floats.EqualApprox(db.WordEnc, file.WordEnc, 0.00000) {
			t.Error(`WordEnc mismatch`, db.WordEnc, file.WordEnc)
		}
		for row := 0; row < db.MFCCRows; row++ {
			if !floats.EqualApprox(db.MFCC[row], file.MFCC[row], 0.00001) {
				t.Error(`MFCC mismatch in row`, row)
			}
		}
	}
	fmt.Println("rows", len(dbRecs), `differences`, rowDiffCount)
}
