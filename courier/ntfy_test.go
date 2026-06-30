package courier

import (
	"context"
	"testing"
)

const topic = "artificial-polyglot"

func TestSendNtfy(t *testing.T) {
	ctx := context.Background()
	err := SendNtfy(ctx, topic, true, "This is test 1", "", "", "")
	if err != nil {
		t.Error(err)
	}
	err = SendNtfy(ctx, topic, false, "This is test 2", "subject", "", "")
	if err != nil {
		t.Error(err)
	}
	err = SendNtfy(ctx, topic, true, "This is test 3", "subject", "5", "")
	if err != nil {
		t.Error(err)
	}
	err = SendNtfy(ctx, topic, false, "This is test 4", "subject", "1", "https://google.com")
	if err != nil {
		t.Error(err)
	}

}
