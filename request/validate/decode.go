package validate

import (
	"context"
	"strings"

	"github.com/artificial-polyglot/arti/request"
)

type RequestValidator struct {
	ctx    context.Context
	errors []string
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
