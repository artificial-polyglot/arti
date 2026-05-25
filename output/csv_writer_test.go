package output

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"gonum.org/v1/gonum/floats"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestCSVWriterScript(t *testing.T) {
	ctx := context.Background()
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	//prepareTimestampAndFMCCData(conn, `ENGWEB`, `ENGWEBN2DA`, t)
	var out = NewOutput(ctx, conn, `TestScripts`, false, false)
	structs, meta := out.PrepareScripts()
	fmt.Println("Loaded Scripts", len(structs))
	filename, status := out.WriteCSV(structs, meta)
	if status != nil {
		t.Error(status)
	}
	mfccFile := readCSVMFCC(filename, t)
	scripts, status := out.LoadScriptStruct(conn)
	if status != nil {
		t.Fatal(status)
	}
	mfccDB := extractScriptMFCC(scripts)
	compare(mfccDB, mfccFile, t)
	fmt.Println("Written CSV", filename)
}

func TestCSVWriterWord(t *testing.T) {
	ctx := context.Background()
	var conn = db.NewDBAdapter(ctx, `ENGWEB_DBPTEXT.db`)
	//prepareTimestampAndFMCCData(conn, `ENGWEB`, `ENGWEBN2DA`, t)
	var out = NewOutput(ctx, conn, `TestWords`, false, false)
	structs, meta := out.PrepareWords()
	fmt.Println("Loaded Scripts", len(structs))
	filename, status := out.WriteCSV(structs, meta)
	if status != nil {
		t.Fatal(status)
	}
	mfccFile := readCSVMFCC(filename, t)
	words, status := out.LoadWordStruct(conn)
	if status != nil {
		t.Fatal(status)
	}
	mfccDB := extractWordEncAndMFCC(words)
	compare(mfccDB, mfccFile, t)
	fmt.Println("Written CSV", filename)
}

func readCSVMFCC(filename string, t *testing.T) [][]float64 {
	var results = make([][]float64, 0, 10000)
	file, _ := os.Open(filename)
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Error(err)
	}
	header := records[0]
	var firstReal int
	for i, col := range header {
		if strings.HasPrefix(col, `mfcc`) || strings.HasPrefix(col, `word_enc`) {
			firstReal = i
			break
		}
	}
	for row, record := range records {
		if row > 0 {
			hasData := false
			tmpRec := record[firstReal:]
			var rec = make([]float64, 0, len(tmpRec))
			for _, val := range tmpRec {
				value, err := strconv.ParseFloat(val, 64)
				if err != nil {
					value = 0.0
				} else {
					hasData = true
				}
				rec = append(rec, value)
			}
			if hasData {
				results = append(results, rec)
			}
		}
	}
	return results
}

func extractScriptMFCC(scripts []Script) [][]float64 {
	var results [][]float64
	for _, scr := range scripts {
		for row := 0; row < scr.MFCCRows; row++ {
			var resultRec []float64
			for _, value := range scr.MFCC[row] {
				resultRec = append(resultRec, float64(value))
			}
			results = append(results, resultRec)
		}
	}
	return results
}

func extractWordEncAndMFCC(words []Word) [][]float64 {
	var results [][]float64
	var wordEncLen = 0
	var mfccCols = 0
	for _, wrd := range words {
		if len(wrd.WordEnc) > 0 {
			wordEncLen = len(wrd.WordEnc)
		}
		if wrd.MFCCCols > 0 {
			mfccCols = wrd.MFCCCols
		}
		if wordEncLen > 0 && mfccCols > 0 {
			break
		}
	}
	for _, wrd := range words {
		if len(wrd.WordEnc) > 0 || len(wrd.MFCC) > 0 {
			var rec = make([]float64, wordEncLen+mfccCols)
			if len(wrd.WordEnc) > 0 {
				for i := 0; i < len(wrd.WordEnc); i++ {
					rec[i] = wrd.WordEnc[i]
				}
			}
			if wrd.MFCCCols > 0 {
				for col := 0; col < wrd.MFCCCols; col++ {
					rec[wordEncLen+col] = float64(wrd.MFCC[0][col])
				}
				results = append(results, rec)
				for row := 1; row < wrd.MFCCRows; row++ {
					rec = make([]float64, wordEncLen+mfccCols)
					for col := 0; col < wrd.MFCCCols; col++ {
						rec[wordEncLen+col] = float64(wrd.MFCC[row][col])
					}
					results = append(results, rec)
				}
			} else {
				results = append(results, rec)
			}
		}
	}
	return results
}

func compare(dbSlice [][]float64, fileSlice [][]float64, t *testing.T) {
	if len(dbSlice) != len(fileSlice) {
		t.Error(`dbSlice has length`, len(dbSlice), `fileSlice has length`, len(fileSlice))
	}
	fmt.Println(`Compare `, len(dbSlice), `records`)
	var rowDiffCount = 0
	for i, dbRec := range dbSlice {
		fileRec := fileSlice[i]
		if len(dbRec) != len(fileRec) {
			t.Error(`row`, i, `dbRec len`, len(dbRec), `fileRec len`, len(fileRec))
		} else if !floats.EqualApprox(dbRec, fileRec, 0.0001) { //0.0000001
			t.Error(`row`, i, `dbRec`, dbRec, `fileRec`, fileRec)
			rowDiffCount++
		}
	}
	fmt.Println("rows", len(dbSlice), `differences`, rowDiffCount)
}
