package validate

import (
	"context"
	"strings"

	"github.com/artificial-polyglot/arti/input/precheck"
	"github.com/artificial-polyglot/arti/request"
)

type RequestValidator struct {
	ctx    context.Context
	errors []string
}

// ValidateAllWASM is the single entry point the WASM build exposes to
// runner_page.js's Save JSON button: it builds the Request from the form's
// JSON dict, validates it, then hands the same *Request to file
// validation so it can read Testament/text_format and populate
// AudioData/TextData.AWSS3 from the dropped directory - all within one Go
// call, so the two validation steps share state as plain pointer passing
// instead of needing any cross-WASM-call mechanism. Only marshals to YAML
// once both steps have had their say.
func ValidateAllWASM(htmlValues string, filePaths string) ([]byte, []string) {
	reqMap := jsonToMap(htmlValues)
	req := createRequestFromMap(reqMap)

	errors := ValidateRequest(context.Background(), &req)
	fileErrors := precheck.ValidateFilesWASM(&req, filePaths, reqMap["text_format_sfm"] == "true")
	errors = append(errors, fileErrors...)

	if len(errors) > 0 {
		return nil, errors
	}
	byts, err := Marshal(req)
	if err != nil {
		errors = append(errors, err.Error())
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
