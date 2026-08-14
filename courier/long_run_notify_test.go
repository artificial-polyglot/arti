package courier

import (
	"context"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request/decode"
	"github.com/artificial-polyglot/arti/request/validate"
	"strings"
	"testing"
	"time"
)

func TestLongRunNotify(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stdout")
	const yamlRequest = `is_new: Y
username: Sam_I_Am
dataset_name: Test_Dataset
language_iso: eng
notify_ok: [gary@shortsands.com, sqs/vessel]
notify_err: [gary@shortsands.com, sqs/vessel]
`
	request, status := decode.Decode(ctx, []byte(yamlRequest))
	if status != nil {
		t.Fatal(status)
	}
	errors := validate.ValidateRequest(ctx, &request)
	if len(errors) > 0 {
		t.Fatal(strings.Join(errors, "\n"))
	}
	// Below this belongs in controller to be in production
	notify := NewLongRunNotify(ctx, request)
	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(notify.Threshold):
			notify.SendEmail()
		case <-done:
			// Job completed before threshold - monitoring done
		}
	}()
	// This above belongs in controller to be in production
	time.Sleep(1 * time.Minute)
}
