package proofing_rpt

import (
	"context"
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/utility/diff"
)

func TestCharFilter1(t *testing.T) {
	ctx := context.Background()
	dbPath := "/Users/gary/Downloads/arti-output_GaryNTest_N2CCPBBS_arti_00016_database_N2CCPBBS.db"
	conn := db.NewDBAdapter(ctx, dbPath)
	verses, status := SelectCharData(conn)
	if status != nil {
		t.Fatal(status)
	}
	status1 := AddASRTranscript(conn, verses)
	if status1 != nil {
		t.Fatal(status1)
	}
	diffFilter := diff.NewCharFilter1(verses)
	diffFilter.RankReplaceCounts()
	diffFilter.DisplayCharFilter()
	newDiff := diffFilter.VerseSliceCompare(verses)
	diffErr := DiffError(newDiff)
	fmt.Println("Final CER", diffErr)
}

func DiffError(dffs [][]diff.Diff) float64 {
	var total int
	var errors int
	for _, df := range dffs {
		cnt := diff.CountDiff(df)
		total += cnt.RefLen
		errors += cnt.Substitutions + cnt.Deletions + cnt.Insertions
	}
	return float64(errors) / float64(total)
}
