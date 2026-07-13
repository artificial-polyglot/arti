package courier

import (
	"context"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
)

func SendNtfy(ctx context.Context, topic string, success bool, message string, subject string,
	priority string, attachment string) *log.Status {
	dest := "https://ntfy.sh/" + topic

	var req *http.Request
	var err error
	if len(message) > 4000 {
		message = message[:4000]
	}
	switch {
	case attachment != "" && isURL(attachment):
		// Remote attachment: ntfy fetches the URL itself, so the body stays free for the message.
		req, err = http.NewRequest("POST", dest, strings.NewReader(message))
		if err != nil {
			return log.Error(ctx, 500, err, "Failed to create ntfy request")
		}
		req.Header.Set("Attach", attachment)
	case attachment != "":
		// Local attachment: the file bytes ARE the body, so the message moves to a header.
		file, err1 := os.Open(attachment)
		if err1 != nil {
			return log.Error(ctx, 500, err1, "Unable to open attachment", attachment)
		}
		defer file.Close()
		req, err = http.NewRequest("PUT", dest, file)
		if err != nil {
			return log.Error(ctx, 500, err, "Failed to create ntfy request")
		}
		req.Header.Set("Filename", filepath.Base(attachment))
		if message != "" {
			safe := strings.ReplaceAll(message, "\r\n", "\n")
			safe = strings.ReplaceAll(safe, "\n", "\\n")
			safe = mime.QEncoding.Encode("utf-8", safe)
			req.Header.Set("Message", safe)
		}
	default:
		req, err = http.NewRequest("POST", dest, strings.NewReader(message))
		if err != nil {
			return log.Error(ctx, 500, err, "Failed to create ntfy request")
		}
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
	nftyAPIToken := os.Getenv("NTFY_API_TOKEN")
	req.Header.Set("Authorization", "Bearer "+nftyAPIToken)
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return log.Error(ctx, 500, err, "Error posting ntfy message.")
	}
	return nil
}

func isURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}
