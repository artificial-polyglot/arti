package qa_align

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
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
func ProcessFAResults(conn db.DBAdapter, request FARequest, jsonData string) *log.Status {
	var asrChars []CharResult
	if err := json.Unmarshal([]byte(jsonData), &asrChars); err != nil {
		return log.Error(conn.Ctx, 500, err, "ProcessFAResults: unmarshal error")
	}
	words, status := selectWords(conn, request.ScriptId)
	if status != nil {
		return status
	}

	wordCharSlices := alignToWords(asrChars, words)

	var scriptText []string
	var scriptBegin float64
	var scriptEnd float64
	var scriptErrorSum float64
	var scriptCharCount int

	for i, w := range words {
		wChars := wordCharSlices[i]
		if len(wChars) == 0 {
			continue
		}
		var wordText string
		var wordBegin, wordEnd, wordErrorSum float64
		wordText, wordBegin, wordEnd, wordErrorSum, status = insertChars(conn, w.wordID, wChars)
		if status != nil {
			return status
		}

		wordScore := wordErrorSum / float64(len(wChars))
		status = insertWords(conn, w.wordID, wordText, wordBegin, wordEnd, wordScore)
		if status != nil {
			return status
		}

		scriptText = append(scriptText, wordText)
		if i == 0 {
			scriptBegin = wordBegin
		}
		scriptEnd = wordEnd
		scriptErrorSum += wordErrorSum
		scriptCharCount += len(wChars)
	}

	var scriptScore float64
	if scriptCharCount > 0 {
		scriptScore = scriptErrorSum / float64(scriptCharCount)
	}

	transcript := strings.Join(scriptText, " ")
	status = insertScript(conn, request.ScriptId, transcript, scriptBegin, scriptEnd, scriptScore)
	if status != nil {
		return status
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

func selectWords(conn db.DBAdapter, scriptID int64) ([]wordRecord, *log.Status) {
	rows, err := conn.DB.Query(
		`SELECT word_id, lower(word) FROM words WHERE ttype = 'W' AND script_id = ? ORDER BY word_id`,
		scriptID,
	)
	if err != nil {
		return nil, log.Error(conn.Ctx, 500, err, "selectWords")
	}
	defer rows.Close()

	var words []wordRecord
	for rows.Next() {
		var w wordRecord
		if err = rows.Scan(&w.wordID, &w.word); err != nil {
			return nil, log.Error(conn.Ctx, 500, err, "selectWords: scan")
		}
		words = append(words, w)
	}
	if err = rows.Err(); err != nil {
		return nil, log.Error(conn.Ctx, 500, err, "selectWords: iterate")
	}
	return words, nil
}

func createQAAlignTables(conn db.DBAdapter) *log.Status {
	var stmts []string
	stmts = append(stmts, "DROP TABLE IF EXISTS chars_qa_align")
	stmts = append(stmts, `CREATE TABLE chars_qa_align (
				word_id INTEGER NOT NULL,
				seq int NOT NULL,
				char TEXT NOT NULL,
				begin_ts REAL NOT NULL,
				end_ts REAL NOT NULL,
				fa_score REAL NOT NULL,
				PRIMARY KEY (word_id, seq))`)
	stmts = append(stmts, "DROP TABLE IF EXISTS words_qa_align")
	stmts = append(stmts, `CREATE TABLE words_qa_align (
				word_id INTEGER NOT NULL,
				word TEXT NOT NULL,
				begin_ts REAL NOT NULL,
				end_ts REAL NOT NULL,
				fa_score REAL NOT NULL,
				PRIMARY KEY (word_id))`)
	stmts = append(stmts, "DROP TABLE IF EXISTS scripts_qa_align")
	stmts = append(stmts, `CREATE TABLE scripts_qa_align (
				script_id INTEGER NOT NULL,
				transcript TEXT NOT NULL,
				begin_ts REAL NOT NULL,
				end_ts REAL NOT NULL,
				fa_score REAL NOT NULL,
				PRIMARY KEY (script_id))`)
	stmts = append(stmts, "DELETE FROM chars_qa_align")
	stmts = append(stmts, "DELETE FROM words_qa_align")
	stmts = append(stmts, "DELETE FROM scripts_qa_align")
	for _, s := range stmts {
		_, err := conn.DB.Exec(s)
		if err != nil {
			return log.Error(conn.Ctx, 500, err, "Create qa_align tables") // is formatted message needed
		}
	}
	return nil
}

// insertCharsV2 inserts one charsV2 row per char for a single word and returns
// the word's begin timestamp, end timestamp, and sum of fa_error values.
func insertChars(conn db.DBAdapter, wordId int64, wordChars []sa.TimedChar) (word string, begin, end, errorSum float64, status *log.Status) {
	stmt, err := conn.DB.Prepare(`INSERT INTO chars_qa_align
		 (word_id, seq, char, begin_ts, end_ts, fa_score)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return "", 0, 0, 0, log.Error(conn.Ctx, 500, err, "insert chars_qa_align: prepare")
	}
	defer stmt.Close()
	begin = wordChars[0].Begin
	var chars []string
	for seq, c := range wordChars {
		_, err = stmt.Exec(wordId, seq, c.Char, c.Begin, c.End, c.Error)
		if err != nil {
			return "", 0, 0, 0, log.Error(conn.Ctx, 500, err, "insert chars_qa_align: word_id", wordId, seq)
		}
		chars = append(chars, c.Char)
		errorSum += c.Error
		end = c.End
	}
	word = strings.Join(chars, "")
	return word, begin, end, errorSum, nil
}

// updateWords sets the timing and FA score on one word row.
func insertWords(conn db.DBAdapter, wordId int64, word string,
	begin, end, score float64) *log.Status {
	_, err := conn.DB.Exec(
		`INSERT INTO words_qa_align (word_id, word, begin_ts, end_ts, fa_score)
			VALUES (?,?,?,?,?)`, wordId, word, begin, end, score)
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "insert words_qa_align: word_id", wordId)
	}
	return nil
}

// updateScript sets the script-level FA score.
func insertScript(conn db.DBAdapter, scriptId int64, transcript string,
	begin float64, end float64, score float64) *log.Status {
	_, err := conn.DB.Exec(
		`INSERT INTO scripts_qa_align(script_id, transcript, begin_ts, end_ts, fa_score) 
			VALUES (?,?,?,?,?)`, scriptId, transcript, begin, end, score)
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "updateScript: script_id", scriptId)
	}
	return nil
}
