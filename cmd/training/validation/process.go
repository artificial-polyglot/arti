package validation

import (
	"context"
	"database/sql"
	"encoding/csv"
	"strings"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/input/precheck"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

const CompareRptCutoff = 2

type PruneSource int

const (
	NotPruned PruneSource = iota
	FAScore
	Silence
	Both
)

func (p PruneSource) String() string {
	switch p {
	case NotPruned:
		return "Not Pruned"
	case FAScore:
		return "FAScore Pruned"
	case Silence:
		return "Silence Pruned"
	case Both:
		return "Both Pruned"
	default:
		return "unknown"
	}
}

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	ctx := database.Ctx
	var output []db.Output
	verses, status := SelectVerses(database)
	if status != nil {
		return output, status
	}
	prunedMap, status := SelectPrunedData(database)
	if status != nil {
		return output, status
	}
	charCntMap, verseCntMap := computeCharLevenshtein(verses, prunedMap)
	validation := computeValidationSet(verses, prunedMap)
	knownErrorList, status1 := ReadAudioErrorData(ctx, req.DatasetName, req.AudioData.AWSS3)
	if status1 != nil {
		return output, status1
	}
	var knownErrors []knownError
	if len(knownErrorList) > 0 {
		faRankMap, status3 := SelectVerseFARank(database)
		if status3 != nil {
			return output, status3
		}
		knownErrors = checkKnownErrors(verses, prunedMap, faRankMap, knownErrorList)
	}
	rptPath, status2 := generateReport(ctx, req.DatasetName, charCntMap, verseCntMap, validation, knownErrors)
	if status2 != nil {
		return output, status2
	}
	var out = db.Output{Component: "Validate", Report: "Validation Report", FilePath: rptPath}
	output = append(output, out)
	return output, nil
}

type prunedVerse struct {
	scriptId   int64
	reason     PruneSource
	minFAScore float64
	reference  generic.VerseRef
}

// faRank is a verse's minimum words.fa_score, and its percentile among all
// verses ranked the same way VerseDataPruner ranks them (lowest = 0). The geo
// fields are the same ranking by minimum word score, where a word's score is the
// geometric mean of its chars.fa_score instead of the arithmetic mean stored in
// words.fa_score; this is an experiment computed only for this report, and
// nothing is written back.
type faRank struct {
	minFAScore    sql.NullFloat64
	percentile    float64
	geoMin        sql.NullFloat64
	geoPercentile float64
}

type knownError struct {
	reference   string
	scriptId    int64
	pruned      PruneSource
	faRank      faRank
	faRankFound bool
	cer         diff.Counts
	largestDiff int
	found       bool
	diffs       []diff.Diff
	knownText   string // the script line text from _Audio_Errors.csv, with {=}/{-} markers
}

func computeCharLevenshtein(verses []generic.Verse2, pruned map[int64]prunedVerse) (map[PruneSource]diff.Counts, map[PruneSource]int) {
	result := map[PruneSource]diff.Counts{NotPruned: {}, FAScore: {}, Silence: {}, Both: {}}
	verseCount := make(map[PruneSource]int)
	for _, vs := range verses {
		cnt := diff.Count(diff.CharLevenshtein(vs.Text(), vs.ASRText))
		reason := pruned[vs.ScriptId].reason // missing verse -> zero value, NotPruned
		total := result[reason]
		total.Add(cnt)
		result[reason] = total
		verseCount[reason]++
	}
	return result, verseCount
}

// validationSet is the union of all pruned verses, whatever the prune source.
// These verses were excluded from training, so they are the validation dataset,
// and its CER is the single value used to compare models and training parameters.
type validationSet struct {
	verses []generic.Verse2
	cer    diff.Counts
}

func computeValidationSet(verses []generic.Verse2, pruned map[int64]prunedVerse) validationSet {
	var result validationSet
	for _, vs := range verses {
		if _, ok := pruned[vs.ScriptId]; !ok {
			continue
		}
		result.verses = append(result.verses, vs)
		result.cer.Add(diff.Count(diff.CharLevenshtein(vs.Text(), vs.ASRText)))
	}
	return result
}

func checkKnownErrors(verses []generic.Verse2, prune map[int64]prunedVerse, faRanks map[int64]faRank, errors []audioError) []knownError {
	var result []knownError
	var refMap = make(map[generic.VerseRef]generic.Verse2)
	for _, vs := range verses {
		refMap[vs.LineRef] = vs
	}
	for _, er := range errors {
		var known knownError
		known.reference = er.reference
		known.knownText = er.text
		ref := generic.NewVerseRef(er.reference)
		vs, found := refMap[ref]
		known.found = found
		if found {
			known.scriptId = vs.ScriptId
			pruned, ok := prune[vs.ScriptId]
			if ok {
				known.pruned = pruned.reason
			} else {
				known.pruned = NotPruned
			}
			known.faRank, known.faRankFound = faRanks[vs.ScriptId]
			known.cer = diff.Count(diff.CharLevenshtein(vs.Text(), vs.ASRText))
			diffs := diff.DiffConvert(vs.Text(), vs.ASRText)
			pair := vs.ToPair()
			pair.Text.Diffs = diffs
			known.largestDiff = pair.LargestLength()
			known.diffs = diffs
		}
		result = append(result, known)
	}
	return result
}

// SelectVerseFARank ranks verses exactly as cmd/training/wav2vec2 VerseDataPruner
// does (min word fa_score, ties by script_id, headings excluded), so a
// percentile at or below pctExcludePruneData means the verse was FA pruned.
// Verses with no chars rows sort last in the geo ranking and get no geo percentile.
func SelectVerseFARank(database db.DBAdapter) (map[int64]faRank, *log.Status) {
	ctx := database.Ctx
	var result = make(map[int64]faRank)
	query := `WITH verse_min AS (
			SELECT s.script_id, MIN(w.fa_score) AS min_fa_score
			FROM scripts s JOIN words w ON w.script_id = s.script_id
			WHERE w.ttype = 'W' AND s.verse_str != '0'
			GROUP BY s.script_id
		),
		word_geo AS (
			SELECT w.script_id, EXP(AVG(LN(MAX(c.fa_score, 1e-12)))) AS geo_score
			FROM words w JOIN chars c ON c.word_id = w.word_id
			WHERE w.ttype = 'W'
			GROUP BY w.word_id
		),
		geo_min AS (
			SELECT script_id, MIN(geo_score) AS min_geo_score
			FROM word_geo
			GROUP BY script_id
		)
		SELECT v.script_id, v.min_fa_score,
			ROW_NUMBER() OVER (ORDER BY v.min_fa_score, v.script_id) AS rank_num,
			COUNT(*) OVER () AS total,
			g.min_geo_score,
			ROW_NUMBER() OVER (ORDER BY g.min_geo_score IS NULL, g.min_geo_score, v.script_id) AS geo_rank_num,
			COUNT(g.min_geo_score) OVER () AS geo_total
		FROM verse_min v LEFT JOIN geo_min g ON g.script_id = v.script_id`
	rows, err := database.DB.Query(query)
	if err != nil {
		return result, log.Error(ctx, 500, err, "Could not execute ", query)
	}
	defer rows.Close()
	for rows.Next() {
		var scriptId int64
		var rank faRank
		var rankNum int
		var total int
		var geoRankNum int
		var geoTotal int
		err = rows.Scan(&scriptId, &rank.minFAScore, &rankNum, &total, &rank.geoMin, &geoRankNum, &geoTotal)
		if err != nil {
			return result, log.Error(ctx, 500, err, `Error scanning in SelectVerseFARank`)
		}
		rank.percentile = float64(rankNum) * 100.0 / float64(total)
		if rank.geoMin.Valid {
			rank.geoPercentile = float64(geoRankNum) * 100.0 / float64(geoTotal)
		}
		result[scriptId] = rank
	}
	err = rows.Err()
	if err != nil {
		return result, log.Error(ctx, 500, err, `Error at end of rows in SelectVerseFARank`)
	}
	return result, nil
}

// SelectVerses is pulling in all kinds of not needed data from the qa_align tables.  The only
// data that is used by this module from this data is script_id, book_id, chapter_num, verse_str,
// text pulled from words table, and transcript from scripts_qa_align.
func SelectVerses(database db.DBAdapter) ([]generic.Verse2, *log.Status) {
	verses, status := proofing_rpt.SelectCharData(database)
	if status != nil {
		return verses, status
	}
	status = proofing_rpt.AddASRTranscript(database, verses)
	if status != nil {
		return verses, status
	}
	return verses, nil
}

// audioError is one row of {dataset}_Audio_Errors.csv: a reference ("MAT 2:3"), the
// script line number, and the script line text with {=} before text that was
// recorded and {-} before text the recording team found missing from the audio.
type audioError struct {
	reference string
	lineNo    string
	text      string
}

func ReadAudioErrorData(ctx context.Context, datasetName string, s3AudioPath string) ([]audioError, *log.Status) {
	var result []audioError
	const scheme = "s3://"
	parts := strings.SplitN(strings.TrimPrefix(s3AudioPath, scheme), "/", 3)
	bucket := parts[0]
	objKey := parts[1] + "/" + datasetName + "_Audio_Errors.csv"
	s3, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		return result, status
	}
	exists, status := s3.ObjectExists(bucket, objKey)
	if status != nil {
		return result, status
	}
	if !exists {
		return result, nil // No _Audio_Errors.csv file is OK.
	}
	bytes, status := s3.GetObject(bucket, objKey)
	if status != nil {
		return result, status
	}
	content := strings.TrimPrefix(string(bytes), "\uFEFF")
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = 3
	records, err := reader.ReadAll()
	if err != nil {
		return result, log.Error(ctx, 500, err, "Could not parse", objKey)
	}
	for _, rec := range records {
		reference := strings.TrimSpace(rec[0])
		bookId, rest, _ := strings.Cut(reference, " ")
		corrected := precheck.IsBookId(bookId) // e.g. recording teams use JMS, the database uses JAS
		if corrected == "" {
			return result, log.ErrorNoErr(ctx, 500, "Invalid book id", bookId, "in", objKey, "reference", reference)
		}
		reference = corrected + " " + rest
		result = append(result, audioError{reference: reference, lineNo: strings.TrimSpace(rec[1]), text: rec[2]})
	}
	return result, nil
}

func SelectPrunedData(database db.DBAdapter) (map[int64]prunedVerse, *log.Status) {
	ctx := database.Ctx
	var result = make(map[int64]prunedVerse)
	query := `SELECT u.script_id, s.book_id, s.chapter_num, s.verse_str, u.reason, u.min_fa_score
		FROM (
    		SELECT script_id, 1 AS reason, COALESCE(min_fa_score, 0) AS min_fa_score FROM pruned_data
    		UNION ALL
    		SELECT script_id, 2, 0 FROM pruned_silence
		) u
		JOIN scripts s ON s.script_id = u.script_id
		ORDER BY u.script_id, u.reason`
	rows, err := database.DB.Query(query)
	if err != nil {
		return result, log.Error(ctx, 500, err, "Could not execute ", query)
	}
	defer rows.Close()
	for rows.Next() {
		var p prunedVerse
		var bookId string
		var chapterNum int
		var verseStr string
		err = rows.Scan(&p.scriptId, &bookId, &chapterNum, &verseStr, &p.reason, &p.minFAScore)
		if err != nil {
			return result, log.Error(ctx, 500, err, `Error scanning in SelectPrunedData`)
		}
		p.reference = generic.VerseRef{BookId: bookId, ChapterNum: chapterNum, VerseStr: verseStr}
		exists, found := result[p.scriptId]
		if found {
			exists.reason = Both
			result[p.scriptId] = exists
		} else {
			result[p.scriptId] = p
		}
	}
	err = rows.Err()
	if err != nil {
		return result, log.Error(ctx, 500, err, `Error at end of rows in SelectPrunedData`)
	}
	return result, nil
}
