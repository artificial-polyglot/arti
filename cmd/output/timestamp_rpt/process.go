package timestamp_rpt

import (
	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func Process(database db.DBAdapter, req request.Request) ([]db.Output, *log.Status) {
	var output []db.Output
	verses, status := proofing_rpt.SelectCharData(database)
	if status != nil {
		return output, status
	}
	output = TimestampWriters(database.Ctx, verses)
	return output, nil
}
