package main

import (
	"syscall/js"

	"github.com/artificial-polyglot/arti/utility/safe"
)

func processRequest(this js.Value, args []js.Value) any {
	return safe.SafeVerseNum(args[0].String())
}

func main() {
	println("Hello World")
	js.Global().Set("ProcessRequest", js.FuncOf(processRequest))
	<-make(chan struct{})
}
