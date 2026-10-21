package compare_rpt

import (
	"strings"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/read"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/artificial-polyglot/arti/utility/uroman"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func CompareReport(conn db.DBAdapter, langISO string) ([]generic.Pair, map[string]generic.AudioFile, *log.Status) {
	var records []generic.Pair
	var fileMap map[string]generic.AudioFile
	verses, status := proofing_rpt.SelectCharData(conn)
	if status != nil {
		return records, fileMap, status
	}
	status = proofing_rpt.AddASRTranscript(conn, verses)
	if status != nil {
		return records, fileMap, status
	}
	urExec, status1 := uroman.NewUromanExec(conn.Ctx, langISO, true)
	if status1 != nil {
		return records, fileMap, status1
	}
	for _, vs := range verses {
		diffs := diff.DiffMatchPatch(vs.Text(), vs.ASRText)
		if !diff.IsMatch(diffs) {
			pair := vs.ToPair()
			diffs = CleanupSpaceAndHyphen(diffs)
			pair.Text.Diffs = diff.Convert2Diff(diffs)
			pair.Text.HTML = diff.DiffPrettyHtml(diffs)
			asrUroman, status2 := urExec.Transliterate(vs.ASRText)
			if status2 != nil {
				return records, fileMap, status2
			}
			diffs = diff.DiffMatchPatch(vs.Uroman(), asrUroman)
			diffs = CleanupSpaceAndHyphen(diffs)
			pair.Uroman.Diffs = diff.Convert2Diff(diffs)
			pair.Uroman.HTML = diff.DiffPrettyHtml(diffs)
			records = append(records, pair)
		}
	}
	fileMap, status = db.CreateAudioFileMap(conn)
	return records, fileMap, status
}

func CleanupSpaceAndHyphen(diffs []diffmatchpatch.Diff) []diffmatchpatch.Diff {
	var result []diffmatchpatch.Diff
	for _, dif := range diffs {
		if dif.Type == diffmatchpatch.DiffEqual {
			result = append(result, dif)
		} else if dif.Type == diffmatchpatch.DiffDelete {
			var part []rune
			chars := []rune(dif.Text)
			for _, ch := range chars {
				if read.IsSpace(ch) {
					if len(part) > 0 {
						result = append(result, diffmatchpatch.Diff{Type: diffmatchpatch.DiffDelete, Text: string(part)})
						part = nil
					}
					result = append(result, diffmatchpatch.Diff{Type: diffmatchpatch.DiffEqual, Text: string(ch)})
				} else {
					part = append(part, ch)
				}
			}
			if len(part) > 0 {
				result = append(result, diffmatchpatch.Diff{Type: diffmatchpatch.DiffDelete, Text: string(part)})
			}
		} else if dif.Type == diffmatchpatch.DiffInsert {
			text := StripSpaceAndHyphen(dif.Text)
			result = append(result, diffmatchpatch.Diff{Type: diffmatchpatch.DiffInsert, Text: text})
		}
	}
	return result
}

func StripSpaceAndHyphen(s string) string {
	return strings.Map(func(r rune) rune {
		if IsHyphen(r) || read.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func IsHyphen(ch rune) bool {
	switch ch {
	case '\u002D', // hyphen-minus
		'\u00AD',                                                   // soft hyphen
		'\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015', // hyphen .. horizontal bar
		'\u2212',                     // minus sign
		'\uFE58', '\uFE62', '\uFE63', // small em dash, small hyphen-minus
		'\uFF0D': // fullwidth hyphen-minus
		return true
	}
	return false
}
