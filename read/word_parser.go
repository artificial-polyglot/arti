package read

import (
	"context"
	"strconv"
	"unicode"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

/**
WordParser reads text scripts from the script table and produces the words table.
It replaces an earlier rune-by-rune implementation (archived in
xxxtobedeleted/word_parser) that had two problems:

 1. Malformed input (e.g. a truncated or invalid verse marker) used to be logged
    and then silently ignored, leaving the finite state machine in a broken loop
    that kept consuming runes without emitting them. WordParser treats that as a
    fatal error: tokenizeScriptText returns it immediately and Parse aborts the
    run without inserting any words, instead of writing partial/wrong output.

 2. Unicode combining marks (category Mn, Mc, Me) - such as the vowel signs and
    viramas used in Bengali and other Indic scripts - are not IsLetter, so a rune
    by rune parse used to treat them as punctuation and split one word into a
    word/punctuation/word sequence. WordParser always glues a combining mark to
    whatever token is currently being built.

Output term types:
W Word: whole words, including internal hyphens/apostrophes and combining marks
P Punctuation: single punctuation characters
S Whitespace: a run of one or more whitespace characters
V Verse number: {n}, or a range such as {n-n}, {n}-{n}, or {n}_{n}, exactly as written

Some Vessel Text scripts (e.g. Kangri N2XNRPMS) also wrap proper names in curly
braces, such as {अब्राह़म}, to flag them for the narrator - a convention
unrelated to verse numbers. tokenizeScriptText only treats '{' as the start of
a verse marker when a digit immediately follows; otherwise the '{' is emitted
as ordinary punctuation and the following rune starts a new token normally.
*/

type wordToken struct {
	ttype    string
	text     []rune
	verseNum int // only meaningful when ttype == `V`: the last number in the marker
}

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
	return w
}

func (w *WordParser) Parse() *log.Status {
	records, status := w.conn.SelectScripts()
	if status != nil {
		return status
	}
	for _, rec := range records {
		tokens, status2 := tokenizeScriptText(w.ctx, rec.ScriptText)
		if status2 != nil {
			return status2
		}
		var verseNum = rec.VerseNum
		for _, tok := range tokens {
			if tok.ttype == `V` {
				verseNum = tok.verseNum
			}
			status2 = w.addWord(rec.ScriptId, verseNum, tok.ttype, tok.text)
			if status2 != nil {
				return status2
			}
		}
	}
	w.conn.DeleteWords()
	status = w.conn.InsertWords(w.records)
	w.records = nil
	return status
}

func (w *WordParser) addWord(scriptId int, verseNum int, ttype string, text []rune) *log.Status {
	if w.lastScriptId != scriptId {
		w.lastScriptId = scriptId
		w.wordSeq = 0
	}
	w.wordSeq += 1
	if ttype == `` || len(text) == 0 {
		return log.ErrorNoErr(w.ctx, 500, "Apparent bug in WordParser.addWord: empty ttype or text for script_id:", scriptId)
	}
	var rec db.Word
	rec.ScriptId = scriptId
	rec.WordSeq = w.wordSeq
	rec.VerseNum = verseNum
	rec.TType = ttype
	rec.Word = string(text)
	w.records = append(w.records, rec)
	return nil
}

// isCombiningMark reports whether r is a Unicode combining mark, i.e. a rune that
// is always rendered attached to the base rune before it (Mn: nonspacing, Mc:
// spacing combining, Me: enclosing). Such marks are never a word boundary by
// themselves - unicode.IsLetter(r) is false for them, so without this check they
// were being misclassified as punctuation.
func isCombiningMark(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me)
}

// tokenizeScriptText splits text into a sequence of W/S/P/V tokens. It is a pure
// function of its input so it can be tested directly, without a database.
func tokenizeScriptText(ctx context.Context, text string) ([]wordToken, *log.Status) {
	const (
		begin = iota
		space
		word
		wordPunct
		verseNum
		inVerseNum
		verseSuffix
		endVerseNum
		nextVerseNum
	)
	var tokens = make([]wordToken, 0, 64)
	var term = make([]rune, 0, 32)
	var punct rune
	var verseDigits []rune
	var verseValue int
	var state = begin

	flushWord := func() {
		if len(term) > 0 {
			tokens = append(tokens, wordToken{ttype: `W`, text: term})
			term = make([]rune, 0, 32)
		}
	}
	flushSpace := func() {
		if len(term) > 0 {
			tokens = append(tokens, wordToken{ttype: `S`, text: term})
			term = make([]rune, 0, 32)
		}
	}
	flushVerse := func() {
		tokens = append(tokens, wordToken{ttype: `V`, text: term, verseNum: verseValue})
		term = make([]rune, 0, 32)
	}
	emitPunct := func(r rune) {
		tokens = append(tokens, wordToken{ttype: `P`, text: []rune{r}})
	}
	parseErr := func(expected string, actual rune) *log.Status {
		return log.ErrorNoErr(ctx, 500, "Expected", expected, "but found", string(actual), "in", text)
	}
	// beginToken classifies tok as if the tokenizer were starting fresh (state
	// begin) and returns the resulting state. It is shared by the begin state
	// and by the verseNum state's fallback when '{' turns out not to open a
	// verse marker.
	beginToken := func(tok rune) int {
		if unicode.IsSpace(tok) {
			term = append(term, tok)
			return space
		} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
			term = append(term, tok)
			return word
		} else if tok == '{' {
			term = append(term, tok)
			return verseNum
		}
		emitPunct(tok)
		return begin
	}

	for _, tok := range text {
		if isCombiningMark(tok) {
			switch state {
			case word:
				term = append(term, tok)
			case wordPunct:
				term = append(term, punct, tok)
				punct = 0
				state = word
			case begin:
				term = append(term, tok)
				state = word
			case space:
				flushSpace()
				term = append(term, tok)
				state = word
			case verseNum:
				// Not a verse marker after all: '{' followed directly by a
				// combining mark can't be a digit, so it's the same
				// non-verse-marker case as beginToken's fallback.
				term = term[:0]
				emitPunct('{')
				term = append(term, tok)
				state = word
			case verseSuffix:
				// glue to the suffix letter, e.g. a combining mark on {31ख}
				term = append(term, tok)
			default: // inVerseNum, endVerseNum, nextVerseNum
				return nil, parseErr("a verse number digit, '{', '}', '-' or '_'", tok)
			}
			continue
		}
		switch state {
		case begin:
			state = beginToken(tok)
		case space:
			if unicode.IsSpace(tok) {
				term = append(term, tok)
			} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
				flushSpace()
				term = append(term, tok)
				state = word
			} else if tok == '{' {
				flushSpace()
				term = append(term, tok)
				state = verseNum
			} else { // punctuation
				flushSpace()
				emitPunct(tok)
				state = begin
			}
		case word:
			if unicode.IsSpace(tok) {
				flushWord()
				term = append(term, tok)
				state = space
			} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
				term = append(term, tok)
			} else if tok == '{' {
				flushWord()
				term = append(term, tok)
				state = verseNum
			} else { // possible mid-word punctuation, e.g. hyphen or apostrophe
				punct = tok
				state = wordPunct
			}
		case wordPunct:
			if unicode.IsSpace(tok) {
				flushWord()
				emitPunct(punct)
				punct = 0
				term = append(term, tok)
				state = space
			} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
				term = append(term, punct, tok)
				punct = 0
				state = word
			} else if tok == '{' {
				flushWord()
				emitPunct(punct)
				punct = 0
				term = append(term, tok)
				state = verseNum
			} else { // a second punctuation mark: the first one does not join the word
				flushWord()
				emitPunct(punct)
				punct = 0
				emitPunct(tok)
				state = begin
			}
		case verseNum: // just consumed '{', a digit must come next
			if unicode.IsDigit(tok) {
				term = append(term, tok)
				verseDigits = []rune{tok}
				state = inVerseNum
			} else {
				// Not a verse marker after all - e.g. a proper name flagged for
				// the narrator as {Name}. Treat the '{' as ordinary punctuation
				// and reprocess tok as if starting fresh.
				term = term[:0]
				emitPunct('{')
				state = beginToken(tok)
			}
		case inVerseNum:
			if unicode.IsDigit(tok) {
				term = append(term, tok)
				verseDigits = append(verseDigits, tok)
			} else if tok == '}' {
				term = append(term, tok)
				verseValue, _ = strconv.Atoi(string(verseDigits))
				verseDigits = nil
				state = endVerseNum
			} else if tok == '-' || tok == '_' {
				// a range within a single marker, e.g. {10-11}: start over on the
				// digits of the second number so verseValue ends up holding it
				term = append(term, tok)
				verseDigits = nil
			} else if unicode.IsLetter(tok) {
				// a split-verse suffix, e.g. {6a} or the Kangri script's {31ख}:
				// the letter doesn't change verseValue, just tags along in term
				verseValue, _ = strconv.Atoi(string(verseDigits))
				term = append(term, tok)
				state = verseSuffix
			} else {
				return nil, parseErr("a digit, letter, '-', '_' or '}'", tok)
			}
		case verseSuffix:
			if tok == '}' {
				term = append(term, tok)
				state = endVerseNum
			} else if unicode.IsLetter(tok) {
				term = append(term, tok)
			} else {
				return nil, parseErr("a letter or '}'", tok)
			}
		case endVerseNum: // just closed '}', may be followed by a range separator
			if tok == '-' || tok == '_' {
				term = append(term, tok)
				state = nextVerseNum
			} else if unicode.IsSpace(tok) {
				flushVerse()
				term = append(term, tok)
				state = space
			} else if unicode.IsLetter(tok) || unicode.IsNumber(tok) {
				flushVerse()
				term = append(term, tok)
				state = word
			} else if tok == '{' {
				flushVerse()
				term = append(term, tok)
				state = verseNum
			} else { // punctuation
				flushVerse()
				emitPunct(tok)
				state = begin
			}
		case nextVerseNum: // just consumed the range separator, '{' must come next
			if tok == '{' {
				term = append(term, tok)
				state = inVerseNum
			} else {
				return nil, parseErr("'{' after range separator", tok)
			}
		default:
			return nil, log.ErrorNoErr(ctx, 500, "tokenizeScriptText: unknown state", state)
		}
	}
	switch state {
	case word:
		flushWord()
	case wordPunct:
		flushWord()
		emitPunct(punct)
	case space:
		flushSpace()
	case endVerseNum:
		flushVerse()
	case begin:
		// nothing pending
	default: // verseNum, inVerseNum, nextVerseNum: a verse marker was left unclosed
		return nil, log.ErrorNoErr(ctx, 500, "tokenizeScriptText: text ended inside an unterminated verse number", text)
	}
	return tokens, nil
}
