package wav2vec2

import (
	"math"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
)

const numPhrases2SilencePrune = 400

type phraseRow struct {
	scriptId  int64
	audioFile string
	wordSeq   int
	beginTS   float64
	endTS     float64
	faScore   float64
	ttype     string
	word      string
}

type phrase struct {
	scriptId     int64
	audioFile    string
	phraseNum    int
	firstWordSeq int
	lastWordSeq  int
	beginTS      float64
	endTS        float64
	text         string
	faScore      float64
}

// PreparePhrases replaces the phrases table, which holds every phrase of the text.  Each phrase
// is one or more words between punctuation, and fa_score is the minimum fa_score of its words,
// because a phrase is only as trustworthy as its worst word.  begin_ts and end_ts include the
// padding used for training.  first_word_seq and last_word_seq identify the words of the phrase.
func PreparePhrases(conn db.DBAdapter) *log.Status {
	ctx := conn.Ctx
	rows, status := selectPhraseRows(conn)
	if status != nil {
		return status
	}
	phrases := padPhrases(mergeSingleWords(groupPhrases(rows)))
	_, err := conn.DB.Exec("DROP TABLE IF EXISTS phrases")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while dropping phrases.")
	}
	_, err = conn.DB.Exec(`CREATE TABLE phrases (
		phrase_id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id INTEGER NOT NULL,
		phrase_num INTEGER NOT NULL,
		first_word_seq INTEGER NOT NULL,
		last_word_seq INTEGER NOT NULL,
		begin_ts REAL NOT NULL,
		end_ts REAL NOT NULL,
		text TEXT NOT NULL,
		fa_score REAL NOT NULL,
		FOREIGN KEY(script_id) REFERENCES scripts(script_id)) STRICT`)
	if err != nil {
		return log.Error(ctx, 500, err, "Error while creating phrases.")
	}
	query := `INSERT INTO phrases (script_id, phrase_num, first_word_seq, last_word_seq, begin_ts, end_ts, text, fa_score)
		VALUES (?,?,?,?,?,?,?,?)`
	tx, err := conn.DB.Begin()
	if err != nil {
		return log.Error(ctx, 500, err, query)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(query)
	if err != nil {
		return log.Error(ctx, 500, err, query)
	}
	defer stmt.Close()
	for _, p := range phrases {
		_, err = stmt.Exec(p.scriptId, p.phraseNum, p.firstWordSeq, p.lastWordSeq, p.beginTS, p.endTS, p.text, p.faScore)
		if err != nil {
			return log.Error(ctx, 500, err, query)
		}
	}
	err = tx.Commit()
	if err != nil {
		return log.Error(ctx, 500, err, query)
	}
	return nil
}

// PhraseDataPruner excludes every phrase of a verse_str '0' line (headings), plus the pctExclude
// percent of the remaining phrases that have the lowest fa_score.
func PhraseDataPruner(conn db.DBAdapter, pctExclude float64) *log.Status {
	ctx := conn.Ctx
	_, err := conn.DB.Exec("DROP TABLE IF EXISTS pruned_data")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while dropping pruned data.")
	}
	query := `CREATE TABLE pruned_data AS
		WITH ranked AS (
			SELECT p.phrase_id, p.fa_score,
				ROW_NUMBER() OVER (ORDER BY p.fa_score, p.phrase_id) AS rank_num
			FROM phrases p JOIN scripts s ON s.script_id = p.script_id
			WHERE s.verse_str != '0'
		)
		SELECT p.phrase_id, NULL AS min_fa_score
		FROM phrases p JOIN scripts s ON s.script_id = p.script_id
		WHERE s.verse_str = '0'
		UNION ALL
		SELECT phrase_id, fa_score FROM ranked
		WHERE rank_num <= (SELECT COUNT(*) FROM ranked) * ? / 100`
	_, err = conn.DB.Exec(query, pctExclude)
	if err != nil {
		return log.Error(ctx, 500, err, "Error while creating pruned data.")
	}
	return nil
}

// PhraseSilencePruner excludes the numPrune phrases with the longest silence between two of their
// words.  Silence between phrases is not counted, because it is not in the audio of any sample.
func PhraseSilencePruner(conn db.DBAdapter, numPrune int) *log.Status {
	ctx := conn.Ctx
	_, err := conn.DB.Exec("DROP TABLE IF EXISTS pruned_silence")
	if err != nil {
		return log.Error(ctx, 500, err, "Error while dropping pruned silence.")
	}
	query := `CREATE TABLE pruned_silence AS
		WITH gaps AS (
			SELECT p.phrase_id, w.word_begin_ts - LAG(w.word_end_ts)
				OVER (PARTITION BY p.phrase_id ORDER BY w.word_seq) AS silence
			FROM phrases p
			JOIN words w ON w.script_id = p.script_id AND w.word_seq BETWEEN p.first_word_seq AND p.last_word_seq
			WHERE w.ttype = 'W' AND w.word_end_ts > w.word_begin_ts
		)
		SELECT phrase_id FROM gaps
		WHERE silence IS NOT NULL
		GROUP BY phrase_id
		ORDER BY MAX(silence) DESC, phrase_id
		LIMIT ?`
	_, err = conn.DB.Exec(query, numPrune)
	if err != nil {
		return log.Error(ctx, 500, err, "Error while creating pruned silence.")
	}
	return nil
}

func selectPhraseRows(conn db.DBAdapter) ([]phraseRow, *log.Status) {
	ctx := conn.Ctx
	var results []phraseRow
	query := `SELECT s.script_id, s.audio_file, w.word_seq, w.word_begin_ts, w.word_end_ts, w.fa_score, w.ttype, w.word
		FROM scripts s JOIN words w ON w.script_id = s.script_id
		WHERE w.ttype IN ('W', 'P')
		ORDER BY w.script_id, w.word_seq`
	rows, err := conn.DB.Query(query)
	if err != nil {
		return results, log.Error(ctx, 500, err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var r phraseRow
		err = rows.Scan(&r.scriptId, &r.audioFile, &r.wordSeq, &r.beginTS, &r.endTS, &r.faScore, &r.ttype, &r.word)
		if err != nil {
			return results, log.Error(ctx, 500, err, "Error scanning in selectPhraseRows")
		}
		results = append(results, r)
	}
	err = rows.Err()
	if err != nil {
		return results, log.Error(ctx, 500, err, "Error at end of rows in selectPhraseRows")
	}
	return results, nil
}

// groupPhrases expects rows in reading order.  Every punctuation token ends a phrase, and so
// does the end of the script.  Mid-word punctuation is part of its word, so it does not break
// a phrase.  Zero-length words (e.g. footnote references) are dropped.
func groupPhrases(rows []phraseRow) []phrase {
	var phrases []phrase
	var current []phraseRow
	flush := func(phraseNum int) {
		words := make([]string, len(current))
		faScore := math.Inf(1)
		for i, w := range current {
			words[i] = w.word
			faScore = min(faScore, w.faScore)
		}
		first, last := current[0], current[len(current)-1]
		phrases = append(phrases, phrase{scriptId: first.scriptId, audioFile: first.audioFile, phraseNum: phraseNum,
			firstWordSeq: first.wordSeq, lastWordSeq: last.wordSeq, beginTS: first.beginTS, endTS: last.endTS,
			text: strings.Join(words, " "), faScore: faScore})
		current = nil
	}
	phraseNum := 0
	for i, r := range rows {
		if r.ttype == "W" && r.endTS > r.beginTS {
			current = append(current, r)
		}
		lastInScript := i+1 == len(rows) || rows[i+1].scriptId != r.scriptId
		if len(current) > 0 && (r.ttype == "P" || lastInScript) {
			phraseNum++
			flush(phraseNum)
		}
		if lastInScript {
			phraseNum = 0
		}
	}
	return phrases
}

// mergeSingleWords joins each single-word phrase to the phrase before or after it in the same
// script, whichever is separated from it by the shorter pause, and renumbers the phrases.
// A script that contains only one word in total stays a single-word phrase.
func mergeSingleWords(phrases []phrase) []phrase {
	var result []phrase
	for start := 0; start < len(phrases); {
		end := start + 1
		for end < len(phrases) && phrases[end].scriptId == phrases[start].scriptId {
			end++
		}
		verse := append([]phrase(nil), phrases[start:end]...)
		i := 0
		for i < len(verse) && len(verse) > 1 {
			if strings.Contains(verse[i].text, " ") {
				i++
				continue
			}
			toPrev := i == len(verse)-1 ||
				(i > 0 && verse[i].beginTS-verse[i-1].endTS <= verse[i+1].beginTS-verse[i].endTS)
			first := i
			if toPrev {
				first = i - 1
			}
			a, b := verse[first], verse[first+1]
			a.lastWordSeq = b.lastWordSeq
			a.endTS = b.endTS
			a.text = a.text + " " + b.text
			a.faScore = min(a.faScore, b.faScore)
			verse[first] = a
			verse = append(verse[:first+1], verse[first+2:]...)
			i = first + 1
		}
		for num := range verse {
			verse[num].phraseNum = num + 1
		}
		result = append(result, verse...)
		start = end
	}
	return result
}

// padPhrases adds generic.AudioPadSec to each side of every phrase, limited to half the gap to
// the neighbouring phrase in the same audio file.  mms_align pads verses by the same rule, so a
// phrase at the start or end of a verse hears the same lead-in or tail-off as the verse that ASR
// is run on.  It must be given every phrase in reading order, including those that will be
// pruned, so that each phrase's neighbours are its real neighbours.
func padPhrases(phrases []phrase) []phrase {
	result := make([]phrase, len(phrases))
	for i, p := range phrases {
		hasPrev := i > 0 && phrases[i-1].audioFile == p.audioFile
		hasNext := i+1 < len(phrases) && phrases[i+1].audioFile == p.audioFile
		var prevEnd, nextBegin float64
		if hasPrev {
			prevEnd = phrases[i-1].endTS
		}
		if hasNext {
			nextBegin = phrases[i+1].beginTS
		}
		p.beginTS = generic.PadBegin(p.beginTS, prevEnd, hasPrev)
		p.endTS = generic.PadEnd(p.endTS, nextBegin, hasNext)
		result[i] = p
	}
	return result
}
