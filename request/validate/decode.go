package validate

import (
	"bytes"
	"context"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"gopkg.in/yaml.v3"
)

type RequestValidator struct {
	ctx    context.Context
	errors []string
}

func NewRequestValidator(ctx context.Context) RequestValidator {
	var r RequestValidator
	r.ctx = ctx
	return r
}

func (r *RequestValidator) Process(yamlRequest []byte) (request.Request, *log.Status) {
	var request request.Request
	var status *log.Status
	request, status = r.Decode(yamlRequest)
	if status != nil {
		return request, status
	}
	r.Validate(&request)
	r.Prereq(&request)
	r.Depend(request)
	if len(r.errors) > 0 {
		status = &log.Status{}
		status.Status = 400
		status.Message = strings.Join(r.errors, "\n")
		return request, status
	}
	request.BibleId = strings.ToUpper(request.BibleId)
	request.LanguageISO = strings.ToLower(request.LanguageISO)
	if len(request.LanguageISO) == 0 && len(request.BibleId) > 3 {
		request.LanguageISO = strings.ToLower(request.BibleId[:3])
	}
	return request, nil
}

func (r *RequestValidator) Decode(requestYaml []byte) (request.Request, *log.Status) {
	var resp request.Request
	reader := bytes.NewReader(requestYaml)
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	err := decoder.Decode(&resp)
	if err != nil {
		return resp, log.Error(r.ctx, 400, err, `Error decoding YAML to request`)
	}
	resp.Testament.BuildBookMaps() // Builds Map for t.HasOT(bookId), t.HasNT(bookId)
	return resp, nil
}

func (r *RequestValidator) Encode(req request.Request) (string, *log.Status) {
	var result string
	d, err := yaml.Marshal(&req)
	if err != nil {
		return result, log.Error(r.ctx, 500, err, `Error encoding request to YAML`)
	}
	result = string(d)
	return result, nil
}
