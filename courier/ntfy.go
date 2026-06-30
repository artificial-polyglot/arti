package courier

import (
	"context"
	"net/http"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
)

func SendNtfy(ctx context.Context, topic string, success bool, message string, subject string,
	priority string, urlAttachment string) *log.Status {
	url := "https://ntfy.sh/" + topic
	req, err := http.NewRequest("POST", url, strings.NewReader(message))
	if err != nil {
		return log.Error(ctx, 500, err, "Failed to create ntfy request")
	}
	if success {
		req.Header.Set("Tags", "tada")
	} else {
		req.Header.Set("Tags", "warning")
	}
	if subject != "" {
		req.Header.Set("Title", subject)
	}
	if priority != "" {
		req.Header.Set("Priority", priority)
	}
	if urlAttachment != "" {
		req.Header.Set("Attachment", urlAttachment)
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return log.Error(ctx, 500, err, "Error posting ntfy message.")
	}
	return nil
}
