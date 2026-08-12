package request

import (
	"bytes"
	"context"

	log "github.com/artificial-polyglot/arti/logger"
	"gopkg.in/yaml.v3"
)

func Decode(ctx context.Context, requestYaml []byte) (Request, *log.Status) {
	var resp Request
	reader := bytes.NewReader(requestYaml)
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	err := decoder.Decode(&resp)
	if err != nil {
		return resp, log.Error(ctx, 400, err, `Error decoding YAML to request`)
	}
	resp.Testament.BuildBookMaps() // Builds Map for t.HasOT(bookId), t.HasNT(bookId)
	return resp, nil
}

func Encode(ctx context.Context, req Request) (string, *log.Status) {
	var result string
	d, err := yaml.Marshal(&req)
	if err != nil {
		return result, log.Error(ctx, 500, err, `Error encoding request to YAML`)
	}
	result = string(d)
	return result, nil
}
