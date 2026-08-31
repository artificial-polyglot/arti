package research

import (
	"cmp"
	"context"
	"os"
	"path/filepath"
	"slices"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

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
	return localPath
}

type proofData struct {
	scriptId int
	ref      generic.VerseRef
	word     string
	faScore  float64
}

func analyzeProofRpt(dbPath string, cutoff float64) map[generic.VerseRef]int {
	var results []proofData
	conn := db.NewDBAdapter(context.Background(), dbPath)
	var query = `SELECT s.script_id, s.book_id, s.chapter_num, s.verse_str, 
			q.word, q.fa_score
			FROM words_qa_align q JOIN words w ON q.word_id = w.word_id
			JOIN scripts s ON s.script_id = w.script_id
			WHERE w.ttype = 'W' AND s.verse_str != '0' AND w.script_id IN (
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
		err = rows.Scan(&ln.scriptId, &ln.ref.BookId, &ln.ref.ChapterNum, &ln.ref.VerseStr,
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
	type key struct {
		scriptId int
		verseRef generic.VerseRef
	}
	var summarize = make(map[key]int)
	for _, prf := range results {
		if prf.faScore < cutoff {
			k := key{scriptId: prf.scriptId, verseRef: prf.ref}
			count, _ := summarize[k]
			summarize[k] = count + 1
		}
	}

	// convert to sortable list
	type sortable struct {
		ref      generic.VerseRef
		scriptId int
		count    int
	}
	var sortables []sortable
	for k, val := range summarize {
		var srt = sortable{ref: k.verseRef, scriptId: k.scriptId, count: val}
		sortables = append(sortables, srt)
	}
	slices.SortFunc(sortables, func(a, b sortable) int {
		return cmp.Or(
			cmp.Compare(b.count, a.count),       // count desc
			cmp.Compare(b.scriptId, a.scriptId), // then scriptId desc
		)
	})

	for _, v := range sortables {
		println(v.ref.Description(), v.scriptId, v.count)
	}

	var cntMap = make(map[generic.VerseRef]int)
	for i, srt := range sortables {
		cntMap[srt.ref] = i + 1
	}
	return cntMap
}

func exit(e error) {
	println(e)
	os.Exit(1)
}
