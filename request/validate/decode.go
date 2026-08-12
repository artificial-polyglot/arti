package validate

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/artificial-polyglot/arti/request"
)

type RequestValidator struct {
	ctx    context.Context
	errors []string
}

func ValidateRequestWASM(htmlValues string) ([]byte, []string) {
	var req request.Request
	var byts []byte
	var err error
	req = CreateRequestFromHTML(htmlValues)
	errors := ValidateRequest(context.Background(), &req)
	if len(errors) == 0 {
		byts, err = json.Marshal(req)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	return byts, errors
}

func ValidateRequest(ctx context.Context, req *request.Request) []string {
	var r RequestValidator
	r.ctx = ctx
	r.Validate(req)
	r.Prereq(req)
	r.Depend(*req)
	if len(r.errors) > 0 {
		return r.errors
	}
	req.BibleId = strings.ToUpper(req.BibleId)
	req.LanguageISO = strings.ToLower(req.LanguageISO)
	if len(req.LanguageISO) == 0 && len(req.BibleId) > 3 {
		req.LanguageISO = strings.ToLower(req.BibleId[:3])
	}
	return r.errors
}
