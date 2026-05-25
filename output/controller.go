package output

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
)

type Output struct {
	ctx         context.Context
	conn        db.DBAdapter
	requestName string
	normalize   bool
	pad         bool
}

func NewOutput(ctx context.Context, conn db.DBAdapter, reqName string, normalize bool, pad bool) Output {
	var o Output
	o.ctx = ctx
	o.conn = conn
	o.requestName = reqName
	o.normalize = normalize
	o.pad = pad
	return o
}

func (o *Output) PrepareScripts() ([]any, []Meta) {
	var script Script
	meta := o.ReflectStruct(script)
	scripts, status := o.LoadScriptStruct(o.conn)
	if status != nil {
		panic(status)
	}
	numMFCC := o.FindNumScriptMFCC(scripts)
	o.SetNumMFCC(&meta, numMFCC)
	if o.normalize {
		scripts = o.NormalizeScriptMFCC(scripts, numMFCC)
	}
	if o.pad {
		scripts = o.PadScriptRows(scripts, numMFCC)
	}
	scriptAny := o.ConvertScriptsAny(scripts)
	meta = o.FindActiveCols(scriptAny, meta)
	o.SetCSVPos(&meta)
	return scriptAny, meta
}

func (o *Output) PrepareWords() ([]any, []Meta) {
	var word Word
	meta := o.ReflectStruct(word)
	words, status := o.LoadWordStruct(o.conn)
	if status != nil {
		panic(status)
	}
	numMFCC := o.FindNumWordMFCC(words)
	o.SetNumMFCC(&meta, numMFCC)
	o.FindNumWordEnc(words, &meta)
	if o.normalize {
		words = o.NormalizeWordMFCC(words, numMFCC)
	}
	if o.pad {
		words = o.PadWordRows(words, numMFCC)
	}
	wordAny := o.ConvertWordsAny(words)
	meta = o.FindActiveCols(wordAny, meta)
	o.SetCSVPos(&meta)
	return wordAny, meta
}
