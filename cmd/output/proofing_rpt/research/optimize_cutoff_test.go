package research

import (
	"cmp"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/match/diff"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

//type Error struct {
//	ref      generic.VerseRef
//	position int
//}

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
	N1SKNSEC.diffDB = "00004"
	N1SKNSEC.proofDB = "00003"
	N1SKNSEC.proofCutoff = 0.01
	dbs = append(dbs, N1SKNSEC)

	for _, tst := range dbs {
		actualErrors := readErrorFile(tst.mediaId)
		// do proof
		proofDB := loadDatabase(tst.mediaId, tst.proofDB)
		proofErrors := analyzeProofRpt(proofDB, tst.proofCutoff)
		proofStats := locateErrors(actualErrors, proofErrors)
		displayStats(tst, "align", proofStats)
		// do compare
		diffDB := loadPairs(tst.mediaId, tst.diffDB)
		diffErrors := analyzeDiffRpt(diffDB)
		diffStats := locateErrors(actualErrors, diffErrors)
		displayStats(tst, "diff", diffStats)
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

func loadDatabase(mediaId string, runNum string) string {
	objectKey := filepath.Join("GaryNTest", mediaId, "arti", runNum, "database", mediaId+".db")
	localPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "GaryNTest", mediaId+".db")
	client, err := s3_datastore.NewS3Client(context.Background())
	if err != nil {
		exit(err)
	}
	err = client.DownloadFile("arti-output", objectKey, localPath)
	if err != nil {
		exit(err)
	}
	//return strings.TrimSuffix(localPath, ".db")
	return localPath
}

func loadPairs(mediaId string, runNum string) string {
	objectKey := filepath.Join("GaryNTest", mediaId, "arti", runNum, "output", mediaId+"_audio_compare.json")
	localPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "GaryNTest", mediaId+"_audio_compare.json")
	client, err := s3_datastore.NewS3Client(context.Background())
	if err != nil {
		exit(err)
	}
	err = client.DownloadFile("arti-output", objectKey, localPath)
	if err != nil {
		exit(err)
	}
	return localPath
}

type proofData struct {
	//scriptId int
	ref     generic.VerseRef
	word    string
	faScore float64
}

func analyzeProofRpt(dbPath string, cutoff float64) map[generic.VerseRef]int {
	var results []proofData
	conn := db.NewDBAdapter(context.Background(), dbPath)
	var query = `SELECT s.book_id, s.chapter_num, s.verse_str, 
			q.word, q.fa_score
			FROM words_qa_align q JOIN words w ON q.word_id = w.word_id
			JOIN scripts s ON s.script_id = w.script_id
			WHERE w.ttype = 'W' AND w.script_id IN (
       				SELECT DISTINCT w2.script_id
       				FROM words w2 JOIN words_qa_align q2 ON w2.word_id = q2.word_id
       				WHERE q2.fa_score <= ?)
				ORDER BY w.word_id`
	rows, err := conn.DB.Query(query, cutoff)
	if err != nil {
		exit(err)
	}
	defer rows.Close()
	for rows.Next() {
		var ln proofData
		err = rows.Scan(&ln.ref.BookId, &ln.ref.ChapterNum, &ln.ref.VerseStr,
			&ln.word, &ln.faScore)
		if err != nil {
			exit(err)
		}
		results = append(results, ln)
	}
	err = rows.Err()
	if err != nil {
		exit(err)
	}
	conn.Close()

	// count the number faScores below cutoff per verse
	var summarize = make(map[generic.VerseRef]int)
	for _, prf := range results {
		if prf.faScore < cutoff {
			count, _ := summarize[prf.ref]
			summarize[prf.ref] = count + 1
		}
	}

	// convert to sortable list
	type sortable struct {
		ref   generic.VerseRef
		count int
	}
	var sortables []sortable
	for ref, cnt := range summarize {
		var srt = sortable{ref: ref, count: cnt}
		sortables = append(sortables, srt)
	}
	slices.SortFunc(sortables, func(a, b sortable) int {
		return cmp.Compare(b.count, a.count) // b, a = descending
	})

	// compute the position
	var cntMap = make(map[generic.VerseRef]int)
	for pos, srt := range sortables {
		cntMap[srt.ref] = pos + 1
	}
	return cntMap
}

func analyzeDiffRpt(dbPath string) map[generic.VerseRef]int {
	bytes, err := os.ReadFile(dbPath)
	if err != nil {
		exit(err)
	}
	var result []diff.Pair
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		exit(err)
	}

	/// Missing a step here, I need to sort by length, and the map position, not length
	var lenMap = make(map[generic.VerseRef]int)
	for _, pair := range result {
		ref := generic.VerseRef{BookId: pair.Ref.BookId,
			ChapterNum: pair.Ref.ChapterNum,
			VerseStr:   pair.Ref.VerseStr}
		lenMap[ref] = pair.LargestLength()
	}
	return lenMap
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

func exit(e error) {
	println(e)
	os.Exit(1)
}
