package courier

import (
	"context"
	"os"
	"path"
	"testing"
)

const topic1 = "artificial-polyglot"
const topic2 = "arti2"

func TestSendNtfy(t *testing.T) {
	ctx := context.Background()
	err := SendNtfy(ctx, topic1, true, "This is test 1", "", "", "")
	if err != nil {
		t.Error(err)
	}
	err = SendNtfy(ctx, topic1, false, "This is test 2", "subject", "", "")
	if err != nil {
		t.Error(err)
	}
	err = SendNtfy(ctx, topic1, true, "This is test 3", "subject", "5", "")
	if err != nil {
		t.Error(err)
	}
	err = SendNtfy(ctx, topic1, false, "This is test 4", "subject", "1", "https://google.com")
	if err != nil {
		t.Error(err)
	}
}

func TestSendNtfyOnce(t *testing.T) {
	ctx := context.Background()
	filePath := path.Join(os.Getenv("HOME"), "temporal.log")
	err := SendNtfy(ctx, topic2, true, "This is test once", "", "", filePath)
	if err != nil {
		t.Error(err)
	}
}
