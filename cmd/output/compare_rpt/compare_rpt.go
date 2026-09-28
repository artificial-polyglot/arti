package compare_rpt

import (
	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
)

func CompareReport(conn db.DBAdapter) ([]generic.Pair, map[string]generic.AudioFile, *log.Status) {
	var records []generic.Pair
	var fileMap map[string]generic.AudioFile
	align := proofing_rpt.NewAlignSilence(conn)
	verses, status := proofing_rpt.SelectCharData(conn)
	if status != nil {
		return records, fileMap, status
	}
	for _, vs := range verses {
		refText := vs.Text()
		asrText, status1 := align.SelectTranscript(vs.ScriptId)
		if status1 != nil {
			return records, fileMap, status1
		}
		diffs := diff.DiffMatchPatch(refText, asrText)
		if !diff.IsMatch(diffs) {
			pair := vs.ToPair(vs.ScriptId, asrText)
			pair.Diffs = diffs
			pair.HTML = diff.DiffPrettyHtml(diffs)
			records = append(records, pair)
		}
	}
	fileMap, status = db.CreateAudioFileMap(conn)
	return records, fileMap, status
}
