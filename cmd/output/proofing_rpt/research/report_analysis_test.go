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
	N1SKNSEC.proofDB = "00004"
	N1SKNSEC.proofCutoff = 0.2
	dbs = append(dbs, N1SKNSEC)
	var N2ATGMLT test
	N2ATGMLT.mediaId = "N2ATGMLT"
	N2ATGMLT.diffDB = "00002"
	N2ATGMLT.proofDB = "00003"
	N2ATGMLT.proofCutoff = 0.2
	//	dbs = append(dbs, N2ATGMLT)
	var N2CCPBBS test
	N2CCPBBS.mediaId = "N2CCPBBS"
	N2CCPBBS.diffDB = "00004"
	N2CCPBBS.proofDB = "00005"
	N2CCPBBS.proofCutoff = 0.2
	//	dbs = append(dbs, N2CCPBBS)
	var N2MGUPNG test
	N2MGUPNG.mediaId = "N2MGUPNG"
	N2MGUPNG.diffDB = "00002"
	N2MGUPNG.proofDB = "00004"
	N2MGUPNG.proofCutoff = 0.2
	//	dbs = append(dbs, N2MGUPNG)
	var N2QAEBSP test
	N2QAEBSP.mediaId = "N2QAEBSP"
	N2QAEBSP.diffDB = "00001"
	N2QAEBSP.proofDB = "00002"
	N2QAEBSP.proofCutoff = 0.2
	//	dbs = append(dbs, N2QAEBSP)
	var N2SHNOMF test
	N2SHNOMF.mediaId = "N2SHNOMF"
	N2SHNOMF.diffDB = "00001"
	N2SHNOMF.proofDB = "00002"
	N2SHNOMF.proofCutoff = 0.2
	//dbs = append(dbs, N2SHNOMF) No test results
	var N2XNRPMS test
	N2XNRPMS.mediaId = "N2XNRPMS"
	N2XNRPMS.diffDB = "00010"
	N2XNRPMS.proofDB = "00010"
	N2XNRPMS.proofCutoff = 0.2
	//	dbs = append(dbs, N2XNRPMS)
	var P2LBEBTI test
	P2LBEBTI.mediaId = "P2LBEBTI"
	P2LBEBTI.diffDB = "00002"
	P2LBEBTI.proofDB = "00002"
	P2LBEBTI.proofCutoff = 0.2
	//	dbs = append(dbs, P2LBEBTI)

	for _, tst := range dbs {
		// do proof
		actualErrors := readErrorFile(tst.mediaId)
		if tst.proofDB != "" {
			proofDB := loadDatabase(tst.mediaId, tst.proofDB)
			proofErrors := analyzeProofRpt(proofDB, tst.proofCutoff)
			_ = locateErrors(tst, "align", actualErrors, proofErrors)
		}
		// do compare
		if tst.diffDB != "" {
			diffDB := loadPairs(tst.mediaId, tst.diffDB)
			diffErrors := analyzeDiffRpt(diffDB)
			_ = locateErrors(tst, "diff", actualErrors, diffErrors)
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
		lin = strings.TrimSpace(lin)
		if len(lin) > 0 && lin[0] != '#' {
			parts := strings.Split(lin, "\t")
			ref := generic.NewVerseRef(parts[0])
			results = append(results, ref)
		}
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

func locateErrors(tst test, typ string, actualErrors []generic.VerseRef, reportErrors map[generic.VerseRef]int) statistic {
	println("Test", tst.mediaId, " ", typ)
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
			//println("Found", ref.Description(), pos)
			positions = append(positions, pos)
			sum += float64(pos)
		}
	}
	if len(positions) > 0 {
		stats.minimum = slices.Min(positions)
		stats.maximum = slices.Max(positions)
		stats.mean = sum / float64(len(positions))
		println("found:", len(positions), " min:", stats.minimum, " max:", stats.maximum, " mean:", stats.mean)
	} else {
		println("None found")
	}
	println()
	return stats
}
