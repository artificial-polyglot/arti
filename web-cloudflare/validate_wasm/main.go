package main

import (
	"syscall/js"

	"github.com/artificial-polyglot/arti/input/precheck"
	"github.com/artificial-polyglot/arti/request/validate"
)

func stringsToJSArray(strs []string) js.Value {
	arr := js.Global().Get("Array").New(len(strs))
	for i, s := range strs {
		arr.SetIndex(i, s)
	}
	return arr
}

func validateRequest(this js.Value, args []js.Value) any {
	htmlValues := args[0].String()
	requestJSON, errors := validate.ValidateRequestWASM(htmlValues)

	result := js.Global().Get("Object").New()
	result.Set("request", string(requestJSON))
	result.Set("errors", stringsToJSArray(errors))
	return result
}

func validateFiles(this js.Value, args []js.Value) any {
	filePathsJSON := args[0].String()
	errors := precheck.ValidateFilesWASM(filePathsJSON)
	return stringsToJSArray(errors)
}

func main() {
	js.Global().Set("ValidateRequest", js.FuncOf(validateRequest))
	js.Global().Set("ValidateFiles", js.FuncOf(validateFiles))
	<-make(chan struct{})
}
