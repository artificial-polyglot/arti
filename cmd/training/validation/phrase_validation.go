package validation

import (
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
)

// phraseData is what a phrase sample training left in the database: every phrase, why it was
// pruned, and which phrase each word belongs to.
type phraseData struct {
	scriptPhrases map[int64][]int64     // script_id -> phrase_ids
	pruned        map[int64]PruneSource // phrase_id -> prune source, unpruned phrases are absent
	wordPhrase    map[int64]int64       // word_id -> phrase_id
}

// UsesPhrasePruning is true when the most recent training pruned phrases.  It looks at
// pruned_data, because a phrases table can remain from an earlier run that used another sample type.
func UsesPhrasePruning(database db.DBAdapter) (bool, *log.Status) {
	var count int
	query := `SELECT COUNT(*) FROM pragma_table_info('pruned_data') WHERE name = 'phrase_id'`
	err := database.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return false, log.Error(database.Ctx, 500, err, query)
	}
	return count > 0, nil
}

func SelectPhraseData(database db.DBAdapter) (phraseData, *log.Status) {
	ctx := database.Ctx
	data := phraseData{scriptPhrases: make(map[int64][]int64), pruned: make(map[int64]PruneSource),
		wordPhrase: make(map[int64]int64)}
	selectPairs := func(query string, handle func(a int64, b int64)) *log.Status {
		rows, err := database.DB.Query(query)
		if err != nil {
			return log.Error(ctx, 500, err, query)
		}
		defer rows.Close()
		for rows.Next() {
			var a, b int64
			err = rows.Scan(&a, &b)
			if err != nil {
				return log.Error(ctx, 500, err, query)
			}
			handle(a, b)
		}
		err = rows.Err()
		if err != nil {
			return log.Error(ctx, 500, err, query)
		}
		return nil
	}
	status := selectPairs(`SELECT script_id, phrase_id FROM phrases ORDER BY phrase_id`, func(scriptId int64, phraseId int64) {
		data.scriptPhrases[scriptId] = append(data.scriptPhrases[scriptId], phraseId)
	})
	if status != nil {
		return data, status
	}
	status = selectPairs(`SELECT phrase_id, 1 FROM pruned_data UNION ALL SELECT phrase_id, 2 FROM pruned_silence`,
		func(phraseId int64, reason int64) {
			if _, found := data.pruned[phraseId]; found {
				data.pruned[phraseId] = Both
			} else {
				data.pruned[phraseId] = PruneSource(reason)
			}
		})
	if status != nil {
		return data, status
	}
	status = selectPairs(`SELECT w.word_id, p.phrase_id FROM phrases p
		JOIN words w ON w.script_id = p.script_id AND w.word_seq BETWEEN p.first_word_seq AND p.last_word_seq
		WHERE w.ttype = 'W'`, func(wordId int64, phraseId int64) {
		data.wordPhrase[wordId] = phraseId
	})
	return data, status
}

// computePhraseLevenshtein returns the same totals as computeCharLevenshtein and
// computeValidationSet, but counted by phrase.  ASR is still of the whole verse, so each edit
// of the verse is assigned to the phrase in which it occurs, and the validation set contains
// only the text of pruned phrases, not the phrases of the same verse that were trained on.
func computePhraseLevenshtein(verses []generic.Verse2, data phraseData) (map[PruneSource]diff.Counts, map[PruneSource]int, validationSet) {
	result := map[PruneSource]diff.Counts{NotPruned: {}, FAScore: {}, Silence: {}, Both: {}}
	phraseCount := make(map[PruneSource]int)
	var validation validationSet
	for _, phraseIds := range data.scriptPhrases {
		for _, phraseId := range phraseIds {
			reason := data.pruned[phraseId] // missing phrase -> zero value, NotPruned
			phraseCount[reason]++
			if reason != NotPruned {
				validation.count++
			}
		}
	}
	for _, vs := range verses {
		for phraseId, cnt := range countByPhrase(vs, data.wordPhrase) {
			reason := data.pruned[phraseId]
			total := result[reason]
			total.Add(cnt)
			result[reason] = total
			if reason != NotPruned {
				validation.cer.Add(cnt)
			}
		}
	}
	return result, phraseCount, validation
}

// countByPhrase assigns each edit of a verse to a phrase.  An inserted char has no place in the
// reference, so it goes to the phrase of the char before it.  The space between two phrases is
// in neither phrase, and an edit of it is not counted.
func countByPhrase(vs generic.Verse2, wordPhrase map[int64]int64) map[int64]diff.Counts {
	owners := phraseOfEachChar(vs, wordPhrase)
	var current int64
	for _, owner := range owners { // an insert before the first char goes to the first phrase
		if owner != 0 {
			current = owner
			break
		}
	}
	buckets := make(map[int64][]diff.CDiff)
	pos := 0
	for _, d := range diff.CharLevenshtein(vs.Text(), vs.ASRText) {
		owner := current
		if d.Type != diff.OpInsert {
			owner = owners[pos]
			pos++
			if owner != 0 {
				current = owner
			}
		}
		if owner != 0 {
			buckets[owner] = append(buckets[owner], d)
		}
	}
	result := make(map[int64]diff.Counts, len(buckets))
	for owner, diffs := range buckets {
		result[owner] = diff.Count(diffs)
	}
	return result
}

// phraseOfEachChar returns the phrase_id of each char of vs.Text(), or 0 for a char in no phrase.
func phraseOfEachChar(vs generic.Verse2, wordPhrase map[int64]int64) []int64 {
	var owners []int64
	for i, wd := range vs.Words {
		phraseId := wordPhrase[wd.WordId]
		if i > 0 {
			var space int64
			if wordPhrase[vs.Words[i-1].WordId] == phraseId {
				space = phraseId
			}
			owners = append(owners, space)
		}
		for range []rune(strings.ToLower(wd.Text)) {
			owners = append(owners, phraseId)
		}
	}
	return owners
}

// addPhraseStatus reports how much of each known error's verse was pruned.  It does not say
// whether the pruned phrases are the ones that contain the error.
func addPhraseStatus(knownErrors []knownError, data phraseData) {
	for i, kn := range knownErrors {
		if !kn.found {
			continue
		}
		phraseIds := data.scriptPhrases[kn.scriptId]
		numPruned := 0
		for _, phraseId := range phraseIds {
			if _, found := data.pruned[phraseId]; found {
				numPruned++
			}
		}
		knownErrors[i].status = strconv.Itoa(numPruned) + " of " + strconv.Itoa(len(phraseIds)) + " phrases pruned"
	}
}
