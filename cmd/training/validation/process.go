package validation

import (
	"bufio"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
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
	knownErrorList, status1 := ReadAudioErrorData(ctx, req.DatasetName, req.AudioData.AWSS3)
	if status1 != nil {
		return output, status1
	}
	faRankMap, status3 := SelectVerseFARank(database)
	if status3 != nil {
		return output, status3
	}
	knownErrors := checkKnownErrors(verses, prunedMap, faRankMap, knownErrorList)
	rptPath, status2 := generateReport(ctx, req.DatasetName, charCntMap, verseCntMap, knownErrors)
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
// verses ranked the same way VerseDataPruner ranks them (lowest = 0).
type faRank struct {
	minFAScore sql.NullFloat64
	percentile float64
}

type knownError struct {
	reference   string
	scriptId    int64
	pruned      PruneSource
	faRank      faRank
	faRankFound bool
	cer         diff.Counts
	largestDiff int
	difference  string
	//rawDiff     []diff.Diff
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

func checkKnownErrors(verses []generic.Verse2, prune map[int64]prunedVerse, faRanks map[int64]faRank, errors []string) []knownError {
	var result []knownError
	var refMap = make(map[generic.VerseRef]generic.Verse2)
	for _, vs := range verses {
		refMap[vs.LineRef] = vs
	}
	for _, er := range errors {
		var known knownError
		known.reference = er
		ref := generic.NewVerseRef(er)
		vs, found := refMap[ref]
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
			known.difference = diff.ReadableDiff(diffs)
			//known.rawDiff = diffs
		} else {
			known.difference = "This verse was not found in data."
		}
		result = append(result, known)
	}
	return result
}

func generateReport(ctx context.Context, datasetName string, charCntMap map[PruneSource]diff.Counts, verseCntMap map[PruneSource]int, knownErrors []knownError) (string, *log.Status) {
	filePath := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), datasetName+"_Validation.txt")
	file, err := os.Create(filePath)
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not create Validation.txt at", filePath)
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	_, _ = w.WriteString("Validation Report for ")
	_, _ = w.WriteString(datasetName)
	_, _ = w.WriteString("\n\n")
	_, _ = w.WriteString("Prune Status\tVerses\tChars\tCER\n")
	for _, reas := range []PruneSource{NotPruned, FAScore, Silence, Both} {
		cnt := charCntMap[reas]
		_, _ = w.WriteString(reas.String())
		_, _ = w.WriteString("\t")
		_, _ = w.WriteString(strconv.Itoa(verseCntMap[reas]))
		_, _ = w.WriteString("\t")
		_, _ = w.WriteString(strconv.Itoa(cnt.RefLen))
		_, _ = w.WriteString("\t")
		_, _ = w.WriteString(strconv.FormatFloat(cnt.ErrorRate(), 'f', 4, 64))
		_, _ = w.WriteString("\n")
	}
	_, _ = w.WriteString("\n\n")
	var caughtCount int
	_, _ = w.WriteString("Known Error\tPrune Status\tMin FA\tFA Pctl\tCER\tLargest\tDiff\n")
	for _, kn := range knownErrors {
		_, _ = w.WriteString(kn.reference)
		_, _ = w.WriteString("\t")
		_, _ = w.WriteString(kn.pruned.String())
		_, _ = w.WriteString("\t")
		if kn.faRank.minFAScore.Valid {
			_, _ = w.WriteString(strconv.FormatFloat(kn.faRank.minFAScore.Float64, 'f', 4, 64))
		}
		_, _ = w.WriteString("\t")
		if kn.faRankFound {
			_, _ = w.WriteString(strconv.FormatFloat(kn.faRank.percentile, 'f', 1, 64))
		}
		_, _ = w.WriteString("\t")
		_, _ = w.WriteString(strconv.FormatFloat(kn.cer.ErrorRate(), 'f', 3, 64))
		_, _ = w.WriteString("\t")
		largest := kn.largestDiff
		if largest > CompareRptCutoff {
			caughtCount++
		}
		_, _ = w.WriteString(strconv.Itoa(largest))
		_, _ = w.WriteString("\t")
		_, _ = w.WriteString(kn.difference)
		_, _ = w.WriteString("\n")
		//_, _ = w.WriteString(fmt.Sprint(kn.rawDiff))
		_, _ = w.WriteString("\n\n")
	}
	missed := len(knownErrors) - caughtCount
	_, _ = w.WriteString("Missed " + strconv.Itoa(missed) + " out of " + strconv.Itoa(len(knownErrors)))
	err = w.Flush()
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not flush Validation.txt")
	}
	return filePath, nil
}

// SelectVerseFARank ranks verses exactly as cmd/training/wav2vec2 VerseDataPruner
// does (min word fa_score, ties by script_id, headings excluded), so a
// percentile at or below pctExcludePruneData means the verse was FA pruned.
func SelectVerseFARank(database db.DBAdapter) (map[int64]faRank, *log.Status) {
	ctx := database.Ctx
	var result = make(map[int64]faRank)
	query := `WITH verse_min AS (
			SELECT s.script_id, MIN(w.fa_score) AS min_fa_score
			FROM scripts s JOIN words w ON w.script_id = s.script_id
			WHERE w.ttype = 'W' AND s.verse_str != '0'
			GROUP BY s.script_id
		)
		SELECT script_id, min_fa_score,
			ROW_NUMBER() OVER (ORDER BY min_fa_score, script_id) AS rank_num,
			(SELECT COUNT(*) FROM verse_min) AS total
		FROM verse_min`
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
		err = rows.Scan(&scriptId, &rank.minFAScore, &rankNum, &total)
		if err != nil {
			return result, log.Error(ctx, 500, err, `Error scanning in SelectVerseFARank`)
		}
		rank.percentile = float64(rankNum) * 100.0 / float64(total)
		result[scriptId] = rank
	}
	err = rows.Err()
	if err != nil {
		return result, log.Error(ctx, 500, err, `Error at end of rows in SelectVerseFARank`)
	}
	return result, nil
}

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

func ReadAudioErrorData(ctx context.Context, datasetName string, s3AudioPath string) ([]string, *log.Status) {
	var result []string
	const scheme = "s3://"
	parts := strings.SplitN(strings.TrimPrefix(s3AudioPath, scheme), "/", 3)
	bucket := parts[0]
	objKey := parts[1] + "/" + datasetName + "_Audio_Errors.txt"
	s3, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		return result, status
	}
	bytes, status1 := s3.GetObject(bucket, objKey)
	if status1 != nil {
		return result, status1
	}
	content := strings.TrimSpace(string(bytes))
	result = strings.Split(content, "\n")
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
