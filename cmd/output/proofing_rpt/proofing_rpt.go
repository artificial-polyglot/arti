package proofing_rpt

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/stdio_exec"
)

type Word struct {
	ScriptId int64
	WordId   int64
	Word     string  // reference word
	Ttype    string  // reference ttype
	BeginTS  float64 // reference TS
	EndTS    float64 // reference TS
	FaScore  float64 // transcript error
	Opacity  float64 // computed from error
}

type Verse struct {
	Ref       generic.VerseRef
	ScriptNum string
	AudioFile string
}

const FA_SCORE_CUTOFF = 0.95

type ProofingRpt struct {
	ctx         context.Context
	conn        db.DBAdapter
	languageISO string
	uRoman      bool
}

func NewProofingRpt(ctx context.Context, conn db.DBAdapter, languageISO string, uRoman bool) ProofingRpt {
	var p ProofingRpt
	p.ctx = ctx
	p.conn = conn
	p.languageISO = languageISO
	p.uRoman = uRoman
	return p
}

func (p *ProofingRpt) Process() ([][]Word, map[int64]Verse, string, *log.Status) {
	var result [][]Word
	var verses map[int64]Verse
	var baseURL string
	words, status := p.SelectWords(FA_SCORE_CUTOFF)
	if status != nil {
		return result, verses, baseURL, status
	}
	if p.uRoman && !p.IsLatin(words) {
		words, status = p.UromanConversion(words, p.languageISO)
		if status != nil {
			return result, verses, baseURL, status
		}
	}
	result = words
	p.computeOpacity(result, FA_SCORE_CUTOFF)
	verses, status = p.findVerseReferences(result)
	if status != nil {
		return result, verses, baseURL, status
	}
	baseURL, status = input.SelectBaseURL(p.conn)
	return result, verses, baseURL, status
}

func (p *ProofingRpt) SelectWords(cutoff float64) ([][]Word, *log.Status) {
	var result [][]Word
	query := `SELECT w.script_id, w.word_id, w.word, w.ttype, w.word_begin_ts, w.word_end_ts, q.fa_score
    FROM words w LEFT OUTER JOIN words_qa_align q ON w.word_id = q.word_id
    WHERE w.script_id IN (
       SELECT DISTINCT w2.script_id
       FROM words w2 JOIN words_qa_align q2 ON w2.word_id = q2.word_id
       WHERE q2.fa_score < :cutoff )
    ORDER BY w.word_id`
	rows, err := p.conn.DB.Query(query, sql.Named("cutoff", cutoff))
	if err != nil {
		return result, log.Error(p.ctx, 500, err, "error querying words with fa_error cutoff")
	}
	defer rows.Close()
	var currentScript int64
	var currentWords []Word
	for rows.Next() {
		var w Word
		var faError sql.NullFloat64
		err = rows.Scan(&w.ScriptId, &w.WordId, &w.Word, &w.Ttype, &w.BeginTS, &w.EndTS,
			&faError)
		if err != nil {
			return result, log.Error(p.ctx, 500, err, "error scanning word row")
		}
		if faError.Valid {
			w.FaScore = faError.Float64
		}
		if w.ScriptId != currentScript {
			if currentWords != nil {
				result = append(result, currentWords)
			}
			currentWords = []Word{}
			currentScript = w.ScriptId
		}
		currentWords = append(currentWords, w)
	}
	if currentWords != nil {
		result = append(result, currentWords)
	}
	if err = rows.Err(); err != nil {
		return result, log.Error(p.ctx, 500, err, "error iterating word rows")
	}
	return result, nil
}

func (c *ProofingRpt) IsLatin(records [][]Word) bool {
	var numChars = 0
	var numLatin = 0
	for _, rec := range records {
		for _, wd := range rec {
			for _, ch := range []rune(wd.Word) {
				numChars++
				if ch <= '\u024F' { // Upper Bound of Latin Extended-B
					numLatin++
				}
			}
		}
	}
	pct := float64(numLatin) / float64(numChars)
	return pct > 0.9 // Is latin is true if 90% of the chars are latin
}

func (p *ProofingRpt) UromanConversion(wordsIn [][]Word, lang string) ([][]Word, *log.Status) {
	words := wordsIn
	scriptPath := filepath.Join(os.Getenv("GOPROJ"), "utility", "uroman", "uroman_stdio.py")
	uromanPkg, status := stdio_exec.NewStdioExec(p.ctx, os.Getenv(`FCBH_MMS_FA_PYTHON`), scriptPath, "-l", lang)
	if status != nil {
		return words, status
	}
	defer uromanPkg.Close()
	var uroman string
	for i := range words {
		for j := range words[i] {
			word := words[i][j].Word
			uroman, status = uromanPkg.Process(word)
			if status != nil {
				return words, status
			}
			words[i][j].Word = matchCapitalization(word, uroman)
		}
	}
	return words, status
}

func matchCapitalization(source, target string) string {
	if source == "" || target == "" {
		return target
	}
	r, _ := utf8.DecodeRuneInString(source)
	if unicode.IsUpper(r) {
		first, size := utf8.DecodeRuneInString(target)
		return string(unicode.ToUpper(first)) + target[size:]
	}
	return target
}

func (p *ProofingRpt) computeOpacity(lines [][]Word, faErrorCutoff float64) {
	for i := range lines {
		for j := range lines[i] {
			if lines[i][j].FaScore < FA_SCORE_CUTOFF {
				lines[i][j].Opacity = 1.0 - lines[i][j].FaScore
			}
		}
	}
}

func (p *ProofingRpt) findVerseReferences(lines [][]Word) (map[int64]Verse, *log.Status) {
	var result = make(map[int64]Verse)
	var scriptIds []string
	for _, ln := range lines {
		if len(ln) > 0 {
			scriptIds = append(scriptIds, strconv.FormatInt(ln[0].ScriptId, 10))
		}
	}
	query := `SELECT script_id, book_id, chapter_num, verse_str, chapter_end, verse_end,
		script_num, audio_file
		FROM scripts WHERE script_id IN (` + strings.Join(scriptIds, ",") + `)`
	rows, err := p.conn.DB.Query(query)
	if err != nil {
		return result, log.Error(p.ctx, 500, err, "error querying verse refs")
	}
	defer rows.Close()
	for rows.Next() {
		var scriptId int64
		var v Verse
		err = rows.Scan(&scriptId, &v.Ref.BookId, &v.Ref.ChapterNum, &v.Ref.VerseStr, &v.Ref.ChapterEnd,
			&v.Ref.VerseEnd, &v.ScriptNum, &v.AudioFile)
		if err != nil {
			return result, log.Error(p.ctx, 500, err, "error scanning verse ref row")
		}
		result[scriptId] = v
	}
	err = rows.Err()
	if err != nil {
		return result, log.Error(p.ctx, 500, err, "error iterating verse ref rows")
	}
	return result, nil
}
