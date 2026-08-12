package request

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
	"gopkg.in/yaml.v3"
)

func Decode(ctx context.Context, requestYaml []byte) (Request, *log.Status) {
	var req Request
	var err error
	reader := bytes.NewReader(requestYaml)
	if IsJSON(requestYaml) {
		decoder := json.NewDecoder(reader)
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&req)
	} else {
		decoder := yaml.NewDecoder(reader)
		decoder.KnownFields(true)
		err = decoder.Decode(&req)
	}
	if err != nil {
		return req, log.Error(ctx, 400, err, `Error decoding YAML/JSON to request`)
	}
	req.Testament.BuildBookMaps() // Builds Map for t.HasOT(bookId), t.HasNT(bookId)
	return req, nil
}

func IsJSON(b []byte) bool {
	b = bytes.TrimSpace(b)
	return len(b) > 0 && (b[0] == '{' || b[0] == '[')
}

func Encode(ctx context.Context, encType string, req Request) (string, *log.Status) {
	var byt []byte
	var err error
	if strings.ToLower(encType) == "json" {
		byt, err = json.Marshal(&req)
	} else {
		byt, err = yaml.Marshal(&req)
	}
	if err != nil {
		return "", log.Error(ctx, 500, err, `Error encoding request to YAML`)
	}
	return string(byt), nil
}
