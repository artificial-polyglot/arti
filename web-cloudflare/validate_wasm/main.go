package main

import (
	"syscall/js"

	"github.com/artificial-polyglot/arti/request/validate"
)

func validateRequest(this js.Value, args []js.Value) any {
	htmlValues := args[0].String()
	requestJSON, errors := validate.ValidateRequestWASM(htmlValues)

	errArr := js.Global().Get("Array").New(len(errors))
	for i, e := range errors {
		errArr.SetIndex(i, e)
	}

	result := js.Global().Get("Object").New()
	result.Set("request", string(requestJSON))
	result.Set("errors", errArr)
	return result
}

func main() {
	js.Global().Set("ValidateRequest", js.FuncOf(validateRequest))
	<-make(chan struct{})
}
