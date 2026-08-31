package research

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/generic"
)

type test struct {
	mediaId     string
	diffDB      string
	proofDB     string
	proofCutoff float64
}

func TestErrorReports(t *testing.T) {
	var dbs []test
	var N1SKNSEC test
	N1SKNSEC.mediaId = "N1SKNSEC"
	N1SKNSEC.diffDB = "00003"
	//N1SKNSEC.proofDB = "00004"
	N1SKNSEC.proofCutoff = 0.01
	dbs = append(dbs, N1SKNSEC)

	for _, tst := range dbs {
		// do proof
		actualErrors := readErrorFile(tst.mediaId)
		if tst.proofDB != "" {
			proofDB := loadDatabase(tst.mediaId, tst.proofDB)
			proofErrors := analyzeProofRpt(proofDB, tst.proofCutoff)
			proofStats := locateErrors(actualErrors, proofErrors)
			displayStats(tst, "align", proofStats)
		}
		// do compare
		if tst.diffDB != "" {
			diffDB := loadPairs(tst.mediaId, tst.diffDB)
			diffErrors := analyzeDiffRpt(diffDB)
			diffStats := locateErrors(actualErrors, diffErrors)
			displayStats(tst, "diff", diffStats)
		}
	}
}

func readErrorFile(mediaId string) []generic.VerseRef {
	var results []generic.VerseRef
	bytes, err := os.ReadFile(mediaId + ".txt")
	if err != nil {
		exit(err)
	}
	lines := strings.Split(string(bytes), "\n")
	for _, lin := range lines {
		parts := strings.Split(lin, "\t")
		ref := generic.NewVerseRef(parts[0])
		results = append(results, ref)
	}
	return results
}

type statistic struct {
	countFound int
	notFound   int
	minimum    int
	maximum    int
	mean       float64
}

func locateErrors(actualErrors []generic.VerseRef, reportErrors map[generic.VerseRef]int) statistic {
	var stats statistic
	var positions []int
	var sum float64
	for _, ref := range actualErrors {
		pos, ok := reportErrors[ref]
		if !ok {
			stats.notFound++
			println("NOT Found", ref.Description())
		} else {
			stats.countFound++
			println("Found", ref.Description(), pos)
			positions = append(positions, pos)
			sum += float64(pos)
		}
	}
	stats.minimum = slices.Min(positions)
	stats.maximum = slices.Max(positions)
	stats.mean = sum / float64(len(positions))
	return stats
}

func displayStats(tst test, typ string, stats statistic) {
	println("Test", tst.mediaId, " ", typ)
	println("min:", stats.minimum, " max:", stats.maximum, " mean:", stats.mean)
	println()
}
