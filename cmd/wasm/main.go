package main

import (
	"syscall/js"

	"github.com/artificial-polyglot/arti/utility/safe"
)

func safeVerseNum(this js.Value, args []js.Value) any {
	return safe.SafeVerseNum(args[0].String())
}

func main() {
	println("Hello World")
	js.Global().Set("SafeVerseNum", js.FuncOf(safeVerseNum))
	<-make(chan struct{})
}
