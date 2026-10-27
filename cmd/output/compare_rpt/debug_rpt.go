package compare_rpt

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func DebugReport(conn db.DBAdapter, req request.Request) *log.Status {
	// These next four lines are repeated in compare_rpt
	verses, status := proofing_rpt.SelectCharData(conn)
	if status != nil {
		return status
	}
	status = proofing_rpt.AddASRTranscript(conn, verses)
	if status != nil {
		return status
	}

	path := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), req.DatasetName+"_debug.txt")
	file, err := os.Create(path)
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "Could not open debug.txt")
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, vs := range verses {
		diffs := diff.DiffMatchPatch(vs.Text(), vs.ASRText)
		if len(diffs) == 0 {
			continue
		} else if len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual {
			continue
		}
		_, _ = w.WriteString(vs.LineRef.Description() + ": ")
		for _, df := range diffs {
			switch df.Type {
			case diffmatchpatch.DiffEqual:
				_, _ = w.WriteString("{=}")
			case diffmatchpatch.DiffInsert:
				_, _ = w.WriteString("{+}")
			case diffmatchpatch.DiffDelete:
				_, _ = w.WriteString("{-}")
			}
			_, _ = w.WriteString(df.Text)
		}
		_, _ = w.WriteString("\n\n")
	}
	err = w.Flush()
	if err != nil { // required: anything still in the buffer is lost without it
		return log.Error(conn.Ctx, 500, err, "Could not flush buffer of debug.txt")
	}
	err = file.Close()
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "Could not flush buffer of debug.txt")
	}
	log.Info(conn.Ctx, "debug_rpt", path)
	return nil
}
