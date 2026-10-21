package train

import (
	"context"
	"sort"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

// DataPruner reduces the data being trained to exclude data that likely includes errors.
// The arti program, also called Artificial Polyglot is being used to proof
// the correctness of audio files, but there is a fundamental problem with this that
// the training might learn the errors, and therefore not be able to identify them as errors.
// To mitigate this problem, this step is using data found during forced alignment to remove
// verses that are likely to have errors.
// The forced alignment is returning a pair of timestamps and a probability of correctness for each
// character.  I summarize these probabilities of correctness to an average for each word,
// and each script line.
// It is my experience that most characters and words that are missing from the audio or in the
// wrong sequence have a probability assigned to them of less that 0.0001.  To make sure
// that all errors are removed, I eliminate more lines that just those.
type WordRec struct {
	wordId  int64
	faScore float64
}

func VerseDataPruner(conn db.DBAdapter) *log.Status {
	ctx := conn.Ctx
	_, err := conn.DB.Exec("DROP TABLE IF EXISTS pruned_data")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while dropping pruned data.")
	}
	query := `CREATE TABLE pruned_data AS
		SELECT script_id
		FROM scripts WHERE verse_str = '0'
		OR fa_score < 0.2
		OR script_id IN
		(SELECT script_id FROM words WHERE ttype='W' AND fa_score < 0.01)`
	_, err = conn.DB.Exec(query)
	if err != nil {
		return log.Error(ctx, 500, err, "Error while creating pruned data.")
	}
	return nil
}

func WordDataPruner(conn db.DBAdapter, pctExclude int) *log.Status {
	ctx := conn.Ctx
	_, err := conn.DB.Exec("DROP TABLE IF EXISTS pruned_data")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while dropping pruned data.")
	}
	_, err = conn.DB.Exec("CREATE TABLE pruned_data (word_id INTEGER NOT NULL, fa_score real NOT NULL)")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while creating pruned data.")
	}
	words, status := selectWords(ctx, conn)
	if status != nil {
		return status
	}
	status = pruneWords(ctx, conn, words, pctExclude)
	return status
}

func selectWords(ctx context.Context, conn db.DBAdapter) ([]WordRec, *log.Status) {
	var results []WordRec
	var query = "SELECT word_id, fa_score FROM words WHERE ttype = 'W'"
	rows, err := conn.DB.Query(query)
	if err != nil {
		return results, log.Error(ctx, 500, err, "Error in SQL query words fa scores")
	}
	defer rows.Close()
	for rows.Next() {
		var w WordRec
		err = rows.Scan(&w.wordId, &w.faScore)
		if err != nil {
			return results, log.Error(ctx, 500, err, `Error scanning in Select word fa scores`)
		}
		results = append(results, w)
	}
	err = rows.Err()
	if err != nil {
		return results, log.Error(ctx, 500, err, `Error at end of rows in Silence`)
	}
	return results, nil
}

func pruneWords(ctx context.Context, conn db.DBAdapter, words []WordRec, pctExcludeWords int) *log.Status {
	sort.Slice(words, func(i, j int) bool {
		return words[i].faScore < words[j].faScore
	})
	cutoff := int(float64(len(words)) * float64(pctExcludeWords) / 100.0)
	query := "INSERT INTO pruned_data (word_id, fa_score) VALUES (?,?)"
	tx, err := conn.DB.Begin()
	if err != nil {
		return log.Error(ctx, 500, err, query)
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		log.Fatal(ctx, err, query)
	}
	defer stmt.Close()
	for i := range cutoff {
		_, err2 := stmt.Exec(words[i].wordId, words[i].faScore)
		if err2 != nil {
			return log.Error(ctx, 500, err2, query)
		}
	}
	err = tx.Commit()
	if err != nil {
		return log.Error(ctx, 500, err, query)
	}
	return nil
}
