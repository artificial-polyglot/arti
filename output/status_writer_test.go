package output

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"
	"testing"
)

func TestDefaultStatus(t *testing.T) {
	ctx := context.Background()
	status := log.Error(ctx, 400, errors.New(`Huh`), `My message`)
	str := status.String()
	if len(str) != 76 {
		t.Error(`Status should be len 76`, len(str))
	}
	var response any
	err := json.Unmarshal([]byte(str), &response)
	if err != nil {
		t.Error(err)
	}
	//fmt.Println(response)
}

func TestJSONStatus(t *testing.T) {
	status, ctx := prepareError(t)
	var out = NewOutput(ctx, db.DBAdapter{}, `TestStatus`, false, false)
	filename, status2 := out.JSONStatus(status, true)
	if status2 != nil {
		t.Fatal(status2)
	}
	fmt.Println(filename)
	if len(filename) != 360 {
		t.Error(`Result should be len 360`, len(filename))
	}
}

func TestCSVStatus(t *testing.T) {
	status, ctx := prepareError(t)
	var out = NewOutput(ctx, db.DBAdapter{}, `TestStatus`, false, false)
	filename, status2 := out.CSVStatus(status, true)
	if len(filename) != 342 {
		t.Error(`Result should be len 342`, len(filename))
	}
	if status2 != nil {
		t.Fatal(status2)
	}
	fmt.Println(filename)
}

func prepareError(t *testing.T) (log.Status, context.Context) {
	var req request.Request
	req.DatasetName = `Test1`
	req.BibleId = `ENGWEB`
	req.IsNew = true
	req.Testament.NT = true
	ctx := context.Background()
	reqDecoder := decode_yaml.NewRequestDecoder(ctx)
	yaml, status := reqDecoder.Encode(req)
	if status != nil {
		t.Error(status)
	}
	ctx = context.WithValue(context.Background(), `request`, yaml)
	err := errors.New("test err")
	status = log.Error(ctx, 400, err, "my error message")
	return *status, ctx
}
