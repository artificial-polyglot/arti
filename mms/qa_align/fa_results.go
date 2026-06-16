package qa_align

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

	sa "github.com/artificial-polyglot/arti/utility/sequence_align"
)

// CharResult holds the per-character forced alignment output from Python.
type CharResult struct {
	Char  string  `json:"char"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Score float64 `json:"score"`
}

// wordRecord is a word row loaded from the database for a given script.
type wordRecord struct {
	wordID int64
	word   string
}

// ProcessFAResults receives the JSON char list from the Python FA module,
// aligns it to the reference words via sequence alignment, inserts rows into
// charsV2, and updates the words and scripts tables.
func ProcessFAResults(db *sql.DB, request FARequest, jsonData string) error {
	var asrChars []CharResult
	if err := json.Unmarshal([]byte(jsonData), &asrChars); err != nil {
		return fmt.Errorf("ProcessFAResults: unmarshal: %w", err)
	}

	words, err := selectWords(db, request.ScriptId)
	if err != nil {
		return err
	}

	wordCharSlices := alignToWords(asrChars, words)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ProcessFAResults: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var scriptErrorSum float64
	var scriptCharCount int

	for i, w := range words {
		wChars := wordCharSlices[i]
		if len(wChars) == 0 {
			continue
		}

		var wordBegin, wordEnd, wordErrorSum float64
		if wordBegin, wordEnd, wordErrorSum, err = insertCharsV2(tx, w.wordID, wChars); err != nil {
			return err
		}

		wordScore := wordErrorSum / float64(len(wChars))
		if err = updateWords(tx, w.wordID, wordBegin, wordEnd, wordScore); err != nil {
			return err
		}

		scriptErrorSum += wordErrorSum
		scriptCharCount += len(wChars)
	}

	var scriptScore float64
	if scriptCharCount > 0 {
		scriptScore = scriptErrorSum / float64(scriptCharCount)
	}

	if err = updateScript(tx, request.ScriptId, scriptScore); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ProcessFAResults: commit: %w", err)
	}
	return nil
}

// alignToWords aligns the ASR char stream to the reference words using
// sequence alignment and returns one []sa.TimedChar per word. Spaces are stripped
// from the ASR stream before alignment. Reference chars with no ASR match
// (deletions) get interpolated timestamps and Error = 1.0.
func alignToWords(asrChars []CharResult, words []wordRecord) [][]sa.TimedChar {
	// Strip spaces — word boundaries come from the reference, not ASR.
	var queryChars []CharResult
	for _, c := range asrChars {
		if c.Char != "|" {
			c.Score = math.Exp(c.Score) // anti natural log
			queryChars = append(queryChars, c)
		}
	}

	// Flatten reference words into a single char sequence, remembering which
	// word and position-within-word each slot belongs to.
	type refPos struct {
		wordIdx int
		r       rune
	}
	var refPositions []refPos
	var refRunes []rune
	for wi, w := range words {
		for _, r := range w.word {
			refPositions = append(refPositions, refPos{wi, r})
			refRunes = append(refRunes, r)
		}
	}

	queryRunes := make([]rune, len(queryChars))
	for i, c := range queryChars {
		rr := []rune(c.Char)
		if len(rr) > 0 {
			queryRunes[i] = rr[0]
		}
	}

	// Run alignment and build a per-ref-position pointer to the matched ASR char.
	matchedQuery := make([]*sa.TimedChar, len(refRunes))
	for _, pair := range sa.Align(refRunes, queryRunes) {
		ri, qi := pair[0], pair[1]
		if ri >= 0 && qi >= 0 {
			c := queryChars[qi]
			matchedQuery[ri] = &sa.TimedChar{Char: c.Char, Begin: c.Start, End: c.End, Error: c.Score}
		}
		// qi >= 0 && ri < 0: ASR insertion with no reference char — discard.
	}

	// Fill in timestamps for deleted reference chars via linear interpolation.
	aligned := sa.InterpolateTimestamps(matchedQuery)

	// Distribute aligned chars back to per-word slices.
	result := make([][]sa.TimedChar, len(words))
	for ri, pos := range refPositions {
		c := aligned[ri]
		c.Char = string(pos.r)
		result[pos.wordIdx] = append(result[pos.wordIdx], c)
	}
	return result
}

func selectWords(db *sql.DB, scriptID int64) ([]wordRecord, error) {
	rows, err := db.Query(
		`SELECT word_id, lower(word) FROM words WHERE ttype = 'W' AND script_id = ? ORDER BY word_id`,
		scriptID,
	)
	if err != nil {
		return nil, fmt.Errorf("selectWords: %w", err)
	}
	defer rows.Close()

	var words []wordRecord
	for rows.Next() {
		var w wordRecord
		if err := rows.Scan(&w.wordID, &w.word); err != nil {
			return nil, fmt.Errorf("selectWords: scan: %w", err)
		}
		words = append(words, w)
		fmt.Print(w)
	}
	fmt.Println()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("selectWords: iterate: %w", err)
	}
	return words, nil
}

func createCharsV2(db *sql.DB) error {
	_, err := db.Exec("DROP TABLE IF EXISTS charsV2")
	if err != nil {
		return fmt.Errorf("drop table charsV2: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS charsV2 (
				word_id INTEGER NOT NULL,
				seq int NOT NULL,
				char TEXT NOT NULL,
				begin_ts REAL NOT NULL,
				end_ts REAL NOT NULL,
				fa_score REAL NOT NULL,
				PRIMARY KEY (word_id, seq))`)
	if err != nil {
		return fmt.Errorf("create charsV2 table: %w", err)
	}
	return nil
}

// insertCharsV2 inserts one charsV2 row per char for a single word and returns
// the word's begin timestamp, end timestamp, and sum of fa_error values.
func insertCharsV2(tx *sql.Tx, wordID int64, wordChars []sa.TimedChar) (begin, end, errorSum float64, err error) {
	stmt, err := tx.Prepare(
		`INSERT INTO charsV2 (word_id, seq, char, begin_ts, end_ts, fa_score)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("insertCharsV2: prepare: %w", err)
	}
	defer stmt.Close()

	begin = wordChars[0].Begin
	for seq, c := range wordChars {
		if _, err = stmt.Exec(wordID, seq, c.Char, c.Begin, c.End, c.Error); err != nil {
			return 0, 0, 0, fmt.Errorf("insertCharsV2: word_id=%d seq=%d: %w", wordID, seq, err)
		}
		errorSum += c.Error
		end = c.End
	}
	return begin, end, errorSum, nil
}

// updateWords sets the timing and FA score on one word row.
func updateWords(tx *sql.Tx, wordID int64, begin, end, score float64) error {
	_, err := tx.Exec(
		`UPDATE words
		 SET word_begin_ts = ?, word_end_ts = ?, fa_score = ?
		 WHERE word_id = ?`,
		begin, end, score, wordID,
	)
	if err != nil {
		return fmt.Errorf("updateWords: word_id=%d: %w", wordID, err)
	}
	return nil
}

// updateScript sets the script-level FA score.
func updateScript(tx *sql.Tx, scriptID int64, score float64) error {
	_, err := tx.Exec(
		`UPDATE scripts SET fa_score = ? WHERE script_id = ?`,
		score, scriptID,
	)
	if err != nil {
		return fmt.Errorf("updateScript: script_id=%d: %w", scriptID, err)
	}
	return nil
}
