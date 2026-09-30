package proofing_rpt

import (
	"context"
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/utility/diff"
)

func TestStudyAlignDiff(t *testing.T) {
	ctx := context.Background()
	dbPath := "/Users/gary/Downloads/arti-output_GaryNTest_N2CCPBBS_arti_00016_database_N2CCPBBS.db"
	conn := db.NewDBAdapter(ctx, dbPath)
	verses, status := SelectCharData(conn)
	if status != nil {
		t.Fatal(status)
	}
	pairs, status1 := CreatePairs(conn, verses)
	if status1 != nil {
		t.Fatal(status1)
	}
	testDiff := PairsCompare(pairs, nil)
	startingCER := DiffError(testDiff)
	fmt.Println("Starting CER", startingCER)
	charCount := CountCharOccurances(pairs)
	countReplace := FindSimilarChars(pairs)
	minCount := 3
	ranked := CreateReplaceCounts(charCount, countReplace, minCount)
	RankReplaceCounts(ranked)
	charReplaceMap := CreateCharReplaceMap(ranked, 5.0)
	for c, m := range charReplaceMap {
		for r, val := range m {
			fmt.Println("From:", string(c), "To:", string(r), val)
		}
	}
	newDiff := PairsCompare(pairs, charReplaceMap)
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
