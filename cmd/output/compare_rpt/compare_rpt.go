package compare_rpt

import (
	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/cmd/speech_to_text/filters"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/artificial-polyglot/arti/utility/uroman"
)

func CompareReport(conn db.DBAdapter, langISO string, sttFilters request.STTFilters) ([]generic.Pair, map[string]generic.AudioFile, filters.Results, *log.Status) {
	var records []generic.Pair
	var fileMap map[string]generic.AudioFile
	verses, status := proofing_rpt.SelectCharData(conn)
	if status != nil {
		return records, fileMap, filters.Results{}, status
	}
	status = proofing_rpt.AddASRTranscript(conn, verses)
	if status != nil {
		return records, fileMap, filters.Results{}, status
	}
	verses, filterResults, status := filters.RunFilters(conn, verses, sttFilters)
	if status != nil {
		return records, fileMap, filterResults, status
	}
	urExec, status1 := uroman.NewUromanExec(conn.Ctx, langISO, true)
	if status1 != nil {
		return records, fileMap, filterResults, status1
	}
	for _, vs := range verses {
		diffs := diff.DiffMatchPatch(vs.Text(), vs.ASRText)
		if !diff.IsMatch(diffs) {
			pair := vs.ToPair()
			pair.Text.Diffs = diff.Convert2Diff(diffs)
			pair.Text.HTML = diff.DiffPrettyHtml(diffs)
			asrUroman, status2 := urExec.Transliterate(vs.ASRText)
			if status2 != nil {
				return records, fileMap, filterResults, status2
			}
			diffs = diff.DiffMatchPatch(vs.Uroman(), asrUroman)
			pair.Uroman.Diffs = diff.Convert2Diff(diffs)
			pair.Uroman.HTML = diff.DiffPrettyHtml(diffs)
			records = append(records, pair)
		}
	}
	fileMap, status = db.CreateAudioFileMap(conn)
	return records, fileMap, filterResults, status
}
