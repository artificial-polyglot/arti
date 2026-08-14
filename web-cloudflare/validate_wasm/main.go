package main

import (
	"syscall/js"

	"github.com/artificial-polyglot/arti/request/validate"
)

func stringsToJSArray(strs []string) js.Value {
	arr := js.Global().Get("Array").New(len(strs))
	for i, s := range strs {
		arr.SetIndex(i, s)
	}
	return arr
}

func validateAll(this js.Value, args []js.Value) any {
	htmlValues := args[0].String()
	filePaths := args[1].String()
	requestYAML, errors := validate.ValidateAllWASM(htmlValues, filePaths)

	result := js.Global().Get("Object").New()
	result.Set("request", string(requestYAML))
	result.Set("errors", stringsToJSArray(errors))
	return result
}

func main() {
	js.Global().Set("ValidateAll", js.FuncOf(validateAll))
	<-make(chan struct{})
}
