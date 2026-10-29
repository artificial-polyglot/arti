package wav2vec2

// Because the DataPreparation step selects words using their begin and end timestamps, this step does not need to identify
// Silence between words or between verses.  However, it does need to identify words that have an unreasonably long
// duration, since that might be another example of silence.  It does this by dividing the duration of each word by its length.
// And then removing those words that exceed some limit, such as 1 or 2 standard deviations above the mean.

import (
	"context"
	"os"
	"strconv"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"gonum.org/v1/gonum/stat"
)

type SilenceWordRec struct {
	BookId          string
	ChapterNum      int
	VerseStr        string
	WordId          int64
	WordLength      int
	BeginTS         float64
	EndTS           float64
	Duration        float64
	DurationPerChar float64
}

func WordSilencePruner(ctx context.Context, conn db.DBAdapter, stdDevOverMean float64) *log.Status {
	_, err := conn.DB.Exec("DROP TABLE IF EXISTS pruned_silence")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while dropping pruned silence.")
	}
	_, err = conn.DB.Exec("CREATE TABLE pruned_silence (word_id INTEGER NOT NULL)")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while creating pruned silence.")
	}
	silences := selectWordSilences(ctx, conn)
	silences = findWordOutliers(ctx, silences, stdDevOverMean)
	query := `INSERT INTO pruned_silence(word_id) VALUES (?)`
	tx, err := conn.DB.Begin()
	if err != nil {
		return log.Error(ctx, 500, err, query)
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		log.Fatal(ctx, err, query)
	}
	defer stmt.Close()
	for _, s := range silences {
		_, err2 := stmt.Exec(s.WordId)
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

func selectWordSilences(ctx context.Context, conn db.DBAdapter) []SilenceWordRec {
	var query = `SELECT s.book_id, s.chapter_num, s.verse_str,
		w.word_id, length(w.word), w.word_begin_ts, w.word_end_ts
		FROM scripts s JOIN words w ON s.script_id = w.script_id
		WHERE w.ttype = 'W' ORDER BY w.word_id`
	rows, err := conn.DB.Query(query)
	if err != nil {
		_ = log.Error(ctx, 500, err, "Error in SQL query of silence")
		os.Exit(1)
	}
	defer rows.Close()
	var results []SilenceWordRec
	for rows.Next() {
		var s SilenceWordRec
		err = rows.Scan(&s.BookId, &s.ChapterNum, &s.VerseStr, &s.WordId, &s.WordLength, &s.BeginTS, &s.EndTS)
		if err != nil {
			_ = log.Error(ctx, 500, err, `Error scanning in Select Silence`)
			os.Exit(1)
		}
		s.Duration = s.EndTS - s.BeginTS
		results = append(results, s)
	}
	err = rows.Err()
	if err != nil {
		_ = log.Error(ctx, 500, err, `Error at end of rows in Silence`)
		os.Exit(1)
	}
	for i := 1; i < len(results)-1; i++ {
		results[i].Duration = results[i].EndTS - results[i].BeginTS
		wordLength := max(results[i].WordLength, 1)
		results[i].DurationPerChar = results[i].Duration / float64(wordLength)
	}
	return results
}

func findWordOutliers(ctx context.Context, silences []SilenceWordRec, stdDevOverMean float64) []SilenceWordRec {
	var durPerChar []float64
	for _, sil := range silences {
		durPerChar = append(durPerChar, sil.DurationPerChar)
	}
	mean := stat.Mean(durPerChar, nil)
	log.Info(ctx, "Mean: ", strconv.FormatFloat(mean, 'f', 2, 64))
	stdDev := stat.StdDev(durPerChar, nil)
	log.Info(ctx, "StdDev: ", strconv.FormatFloat(stdDev, 'f', 2, 64))
	cutoff := mean + stdDevOverMean*stdDev
	var results []SilenceWordRec
	for _, sil := range silences {
		if sil.DurationPerChar > cutoff {
			results = append(results, sil)
		}
	}
	return results
}
