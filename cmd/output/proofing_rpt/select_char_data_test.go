package proofing_rpt

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/utility/diff"
)

func TestSelectCharData(t *testing.T) {
	var countPerfect int
	ctx := context.Background()
	//database := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), "GaryNTest", "N1SKNSEC.db")
	database := filepath.Join(os.Getenv("HOME"), "Downloads", "arti-output_GaryNTest_N2CCPBBS_arti_00012_database_N2CCPBBS.db")
	conn := db.NewDBAdapter(ctx, database)
	align := NewAlignSilence(conn)
	verses, status := SelectCharData(conn)
	if status != nil {
		t.Fatal(status)
	}
	var insertCount = make(map[rune]int)
	var deleteCount = make(map[rune]int)
	var combined = make(map[rune]int)
	for _, vs := range verses {
		refText := align.GetOriginalText(vs)
		asrText, status1 := align.SelectTranscript(vs.ScriptId)
		if status1 != nil {
			t.Fatal(status1)
		}
		cdiffs := diff.CharLevenshtein(refText, asrText)
		cnt := diff.Count(cdiffs)
		if cnt.ErrorRate() > 0 {
			diffs := diff.DiffConvert(refText, asrText)
			for _, df := range diffs {
				if df.Type == diff.OpInsert {
					for _, c := range df.Text {
						insertCount[c]++
						combined[c]++
					}
				} else if df.Type == diff.OpDelete {
					for _, c := range df.Text {
						deleteCount[c]++
						combined[c]++
					}
				}
			}
			//fmt.Println(vs.LineRef.Description(), diffs)
		} else {
			countPerfect++
		}
	}
	fmt.Println("Total Verses:", len(verses), "Perfect:", countPerfect)
	fmt.Println("COMBINED")
	sortMap(combined)
	fmt.Println("DELETED")
	sortMap(deleteCount)
	fmt.Println("INSERTED")
	sortMap(insertCount)
}

func sortMap(counts map[rune]int) {
	type kv struct {
		Key   rune
		Count int
	}
	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	slices.SortFunc(pairs, func(a, b kv) int {
		return cmp.Or(
			cmp.Compare(b.Count, a.Count), // swap a/b for ascending
			cmp.Compare(a.Key, b.Key),
		)
	})
	for _, p := range pairs {
		fmt.Printf("%#U   %d\n", p.Key, p.Count)
	}
}
