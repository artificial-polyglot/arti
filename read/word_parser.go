package read

import (
	"context"
	"strconv"
	"unicode"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

/**
This program reads text scripts from the script table and produces the words table.
It outputs the text as term types and terms.  The following are the term types:
W Word: this is whole words, including hypenated words
P Punctuation: this is single punctuation characters
S Whitespace: this is exact whitespace, and can be multiple characters.
V Verse number: {n} or {n}_{n} this is the exact character string as shown.
While languages might differ, this program needs a consistent way to decide what is a word.
It assumes that whitespace is always a word delimiter, but a single character of punctuation
is not a word delimiter.  But, it assumes that multiple characters of punctuation within a word
is a word delimiter.
*/

type WordParser struct {
	ctx          context.Context
	conn         db.DBAdapter
	wordSeq      int
	lastScriptId int
	records      []db.Word
}

func NewWordParser(conn db.DBAdapter) WordParser {
	var w WordParser
	w.ctx = conn.Ctx
	w.conn = conn
	w.wordSeq = 0
	w.lastScriptId = 0
	return w
}

func (w *WordParser) Parse() *log.Status {
	const (
		begin = iota + 1
		space
		word
		wordPunct
		verseNum
		inVerseNum
		endVerseNum
		nextVerseNum
	)
	var label = []string{"", "BEGIN", "SPACE", "WORD", "WORDPUNCT", "VERSENUM", "INVERSENUM", "ENDVERSENUM",
		"NEXTVERSENUM"}
	var records, status = w.conn.SelectScripts()
	if status != nil {
		return status
	}
	for _, rec := range records {
		//fmt.Printf("%s %d:%s  %s\n", rec.BookId, rec.ChapterNum, rec.VerseStr, rec.ScriptText)
		var term = make([]rune, 0, 100) // None
		var punct rune                  // None
		var verseStr []rune             // None
		var state = begin
		for _, tok := range rec.ScriptText {
			//fmt.Printf("%s\t%d\t%c\t%d\n", label[state], pos, tok, tok)
			switch state {
			case begin:
				if unicode.IsSpace(tok) {
					term = append(term, tok)
					state = space
				} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
					term = append(term, tok)
					state = word
				} else if tok == '{' {
					term = append(term, tok)
					state = verseNum
				} else { // IsPunct
					w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{tok})
					term = []rune{} // redundant
					state = begin
				}
			case space:
				if unicode.IsSpace(tok) {
					term = append(term, tok)
				} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
					w.addWord(rec.ScriptId, rec.VerseNum, `S`, term)
					term = []rune{tok}
					state = word
				} else if tok == '{' {
					w.addWord(rec.ScriptId, rec.VerseNum, `S`, term)
					term = []rune{tok}
					state = verseNum
				} else { // IsPunct
					w.addWord(rec.ScriptId, rec.VerseNum, `S`, term)
					w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{tok})
					term = []rune{}
					state = begin
				}
			case word:
				if unicode.IsSpace(tok) {
					w.addWord(rec.ScriptId, rec.VerseNum, `W`, term)
					term = []rune{tok}
					state = space
				} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
					term = append(term, tok)
				} else if tok == '{' {
					w.addWord(rec.ScriptId, rec.VerseNum, `W`, term)
					term = []rune{tok}
					state = verseNum
				} else { // IsPunct
					punct = tok
					state = wordPunct
				}
			case wordPunct:
				if unicode.IsSpace(tok) {
					w.addWord(rec.ScriptId, rec.VerseNum, `W`, term)
					w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{punct})
					punct = -1
					term = []rune{tok}
					state = space
				} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
					term = append(term, punct)
					term = append(term, tok)
					punct = -1
					state = word
				} else { // IsPunct
					w.addWord(rec.ScriptId, rec.VerseNum, `W`, term)
					w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{punct})
					w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{tok})
					term = []rune{}
					state = begin
				}
			case verseNum:
				if unicode.IsDigit(tok) {
					term = append(term, tok)
					verseStr = []rune{tok}
					state = inVerseNum
				} else {
					w.logError("number", tok)
				}
			case inVerseNum:
				if unicode.IsDigit(tok) {
					term = append(term, tok)
					verseStr = append(verseStr, tok)
				} else if tok == '}' {
					term = append(term, tok)
					rec.VerseNum, _ = strconv.Atoi(string(verseStr))
					verseStr = []rune{}
					state = endVerseNum
				} else {
					w.logError("number or }", tok)
				}
			case endVerseNum:
				if tok == '_' {
					term = append(term, tok)
					state = nextVerseNum
				} else if unicode.IsSpace(tok) {
					w.addWord(rec.ScriptId, rec.VerseNum, `V`, term)
					term = []rune{tok}
					state = space
				} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
					w.addWord(rec.ScriptId, rec.VerseNum, `V`, term)
					term = []rune{tok}
					state = word
				} else { // IsPunct
					w.addWord(rec.ScriptId, rec.VerseNum, `V`, term)
					w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{tok})
					term = []rune{}
					state = begin
				}
			case nextVerseNum:
				if tok == '{' {
					term = append(term, tok)
					state = inVerseNum
				} else {
					w.logError("{", tok)
				}
			default:
				return log.ErrorNoErr(w.ctx, 500, "unknown state", label[state])
			}
		}
		if len(term) > 0 && term[0] != -1 {
			var first = term[0]
			if unicode.IsSpace(first) {
				w.addWord(rec.ScriptId, rec.VerseNum, `S`, term)
			} else if unicode.IsLetter(first) || unicode.IsNumber(first) {
				w.addWord(rec.ScriptId, rec.VerseNum, `W`, term)
			} else { // IsPunct
				w.addWord(rec.ScriptId, rec.VerseNum, `P`, term)
			}
		}
		if state == wordPunct && punct != -1 {
			w.addWord(rec.ScriptId, rec.VerseNum, `P`, []rune{punct})
		}
	}
	w.conn.DeleteWords()
	status = w.conn.InsertWords(w.records)
	w.records = []db.Word{}
	return status
}

func (w *WordParser) addWord(scriptId int, verseNum int, ttype string, text []rune) *log.Status {
	var status *log.Status
	if w.lastScriptId != scriptId {
		w.lastScriptId = scriptId
		w.wordSeq = 0
	}
	w.wordSeq += 1
	if ttype == `` || len(text) == 0 { // or rec.VerseNum == None:
		return log.ErrorNoErr(w.ctx, 500, 0, "Apparent bug in addWord: empty ttype or text for script_id:", scriptId)
	}
	//fmt.Println("seq: ", w.wordSeq, " verse: ", verseNum, " type: ", ttype, " word: ", string(text))
	var rec db.Word
	rec.ScriptId = scriptId
	rec.WordSeq = w.wordSeq
	rec.VerseNum = verseNum
	rec.TType = ttype
	rec.Word = string(text)
	w.records = append(w.records, rec)
	return status
}

func (w *WordParser) logError(expected string, actual rune) *log.Status {
	return log.ErrorNoErr(w.ctx, 500, "Expected: ", expected, ", but found: ", string(actual))
}
