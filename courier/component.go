package courier

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/request/validate"
)

type Component struct {
	ctx      context.Context
	courier  Courier
	start    time.Time
	req      request.Request
	database db.DBAdapter
}

func NewComponent(request string, name string) Component {
	var c Component
	c.ctx = context.Background()
	c.courier = NewCourier(c.ctx, []byte(request))
	c.courier.Component = name
	c.start = time.Now()
	return c
}

func (c *Component) StartComponent() (db.DBAdapter, *log.Status) {
	var status *log.Status
	c.ctx = context.WithValue(c.ctx, "runType", c.courier.Component)
	c.ctx = context.WithValue(c.ctx, `request`, c.courier.yamlContent)
	c.req, status = request.Decode(c.ctx, []byte(c.courier.yamlContent))
	if status != nil {
		return c.database, status
	}
	errors := validate.ValidateRequest(c.ctx, &c.req)
	if len(errors) > 0 {
		return c.database, log.ErrorNoErr(c.ctx, 400, strings.Join(errors, "\n"))
	}
	if !c.req.IsNew {
		dbPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), c.req.Username, c.req.DatasetName+".db")
		if c.req.Database.AWSS3 != "" {
			status = input.DownloadFile(c.ctx, c.req.Database.AWSS3, dbPath)
			if status != nil {
				return c.database, status
			}
		} else if c.req.Database.File != "" {
			err := os.Rename(c.req.Database.File, dbPath)
			if err != nil {
				return c.database, log.Error(c.ctx, 500, err, "Could not move the database file.")
			}
		}
	}
	c.database, status = db.NewerDBAdapter(c.ctx, c.req.IsNew, c.req.Username, c.req.DatasetName)
	if status != nil {
		return c.database, status
	}
	status = c.database.InsertRequest(c.req)
	if status != nil {
		return c.database, status
	}
	c.courier.AddDatabase(c.database)
	return c.database, nil
}

func (c *Component) FinishComponent(outputs []db.Output, runStatus *log.Status) {
	for _, out := range outputs {
		c.courier.AddOutput(out.FilePath)
	}
	_ = c.courier.PersistToBucket(runStatus)                          // do not propagate error
	_ = c.courier.Notification(c.req, runStatus, time.Since(c.start)) // do not propagate error
}
