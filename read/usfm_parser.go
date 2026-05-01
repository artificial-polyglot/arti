package read

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

type USFMParser struct {
	ctx  context.Context
	conn db.DBAdapter
	// transient variables
	styleMap   map[string]USFMStyle
	titleDesc  titleDesc
	stack      USFMStack
	bookId     string
	chapterNum int
	verseStr   string
	verseNum   int
	usfmStyle  string
	text       []string
	skipUntil  string
	scripts    []db.Script
	testOut    *os.File
}

func NewUSFMParser(conn db.DBAdapter) USFMParser {
	var p USFMParser
	p.ctx = conn.Ctx
	p.conn = conn
	p.styleMap = p.BuildUSFMMap()
	p.testOut, _ = os.Create("out.txt")
	return p
}

func (p *USFMParser) ProcessFiles(inputFiles []input.InputFile) *log.Status {
	var status *log.Status
	for _, file := range inputFiles {
		filename := filepath.Join(file.Directory, file.Filename)
		var records []db.Script
		var titles titleDesc
		records, titles, status = p.decode(filename, file.BookId)
		if status != nil {
			return status
		}
		records = p.addChapterHeading(records, titles)
		status = p.conn.InsertScripts(records)
		if status != nil {
			return status
		}
	}
	return status
}

type USFMStyle struct {
	StyleType string // "book", "chapter", "verse", "para", "char", "note", etc.
	Keep      bool
}

func (p *USFMParser) BuildUSFMMap() map[string]USFMStyle {
	result := make(map[string]USFMStyle)
	for key, keep := range usfm {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			continue
		}
		styleType := parts[0]
		code := parts[1]
		result[code] = USFMStyle{
			StyleType: styleType,
			Keep:      keep,
		}
	}
	return result
}

func (p *USFMParser) decode(filename string, bookId string) ([]db.Script, titleDesc, *log.Status) {
	p.titleDesc = titleDesc{}
	p.stack = USFMStack{}
	p.bookId = bookId
	p.chapterNum = 1
	p.verseStr = "0"
	p.verseNum = 0
	p.usfmStyle = ""
	p.text = nil
	p.skipUntil = ""
	p.scripts = nil
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, p.titleDesc, log.Error(p.ctx, 500, err, "Failed to read USFM file")
	}
	const BEGIN = 1
	const SLASH = 2
	const STYLE = 3
	const STYLENUM = 4
	const ENDSTYLE = 5
	const TEXT = 6
	var style string
	var styleNum string
	var text []rune
	var state = BEGIN
	for _, ch := range string(content) {
		//fmt.Printf("state: %d, %s\n", state, char)
		//_, _ = fmt.Fprintln(p.testOut, "state:", state, "style:", style+styleNum, "text:", string(text), "ch:", string(ch), "[]text:", p.text)
		switch state {
		case BEGIN:
			if ch == '\\' {
				state = SLASH
			}
			// no error allows the file to start without a \ at the beginning
		case SLASH:
			if unicode.IsLetter(ch) {
				state = STYLE
				style = string(ch)
				styleNum = ""
			} else {
				return nil, p.titleDesc, log.ErrorNoErr(p.ctx, 500, "Backslash, but no style")
			}
		case STYLE:
			if unicode.IsLetter(ch) {
				state = STYLE
				style += string(ch)
			} else if unicode.IsDigit(ch) {
				state = STYLENUM
				styleNum = string(ch)
			} else if unicode.IsSpace(ch) {
				state = TEXT
				p.stack.Push(style, styleNum, false)
				text = []rune{}
			} else if ch == '*' {
				state = ENDSTYLE
				text = []rune{}
				p.stack.Push(style, styleNum, true)

			} else {
				return nil, p.titleDesc, log.ErrorNoErr(p.ctx, 500, "Failed to read style ")
			}
		case STYLENUM:
			if unicode.IsSpace(ch) {
				state = TEXT
				p.stack.Push(style, styleNum, false)
				text = []rune{}
			} else if ch == '*' {
				state = ENDSTYLE
				p.stack.Push(style, styleNum, true)
				text = []rune{}
			} else {
				return nil, p.titleDesc, log.ErrorNoErr(p.ctx, 500, "failed to read USFM file")
			}
		case TEXT:
			if ch == '\\' {
				state = SLASH
				err = p.storeRecord(text)
				if err != nil {
					return nil, p.titleDesc, log.Error(p.ctx, 500, err)
				}
				text = []rune{}
			} else { // state = TEXT
				text = append(text, ch)
			}
		case ENDSTYLE:
			if ch == '\\' {
				state = SLASH
				err = p.storeRecord([]rune{})
				if err != nil {
					return nil, p.titleDesc, log.Error(p.ctx, 500, err)
				}
			} else {
				state = TEXT
				text = []rune{ch}
			}
		}
	}
	if len(text) > 0 {
		err = p.storeRecord(text)
		if err != nil {
			return nil, p.titleDesc, log.Error(p.ctx, 500, err)
		}
	}
	p.flushPendingVerse()
	return p.scripts, p.titleDesc, nil
}

func (p *USFMParser) flushPendingVerse() {
	if p.verseStr != "" && p.verseStr != "0" {
		rec := db.Script{
			DatasetId:   1,
			BookId:      p.bookId,
			ChapterNum:  p.chapterNum,
			VerseStr:    p.verseStr,
			VerseNum:    p.verseNum,
			UsfmStyle:   p.usfmStyle, //"v",
			ScriptTexts: p.text,
		}
		p.scripts = append(p.scripts, rec)
		p.text = nil
	}
}

func (p *USFMParser) storeRecord(text []rune) error {
	var err error
	fullStyle, ok := p.stack.Pop()
	if !ok {
		if p.skipUntil == "" {
			if whole := strings.TrimSpace(string(text)); whole != "" {
				//p.text = append(p.text, whole)
				p.text = append(p.text, string(text))
			}
		}
		return nil
	}
	style := fullStyle.Style
	styleNum := fullStyle.StyleNum
	//fmt.Printf("style: %s, styleNum: %s\n", style, styleNum)
	usfmStyle := p.styleMap[style]
	switch usfmStyle.StyleType {
	case "book":
		return nil
	case "chapter":
		p.flushPendingVerse()
		p.chapterNum, err = strconv.Atoi(strings.TrimSpace(string(text)))
		if err != nil {
			return err
		}
		p.verseStr = "0"
	case "verse":
		p.flushPendingVerse()
		var wsRegEx = regexp.MustCompile(`\s+`)
		whole := strings.TrimSpace(string(text))
		//whole := strings.TrimLeft(string(text), " \t\n\r")
		parts := wsRegEx.Split(whole, 2)
		p.verseStr = parts[0]
		p.text = nil
		if len(parts) > 1 {
			p.text = []string{parts[1]}
		}
		startVerse := strings.Split(p.verseStr, "-")
		p.verseNum, err = strconv.Atoi(startVerse[0])
		if err != nil {
			return err
		}
	default:
		if style == "h" {
			p.titleDesc.heading = strings.TrimSpace(string(text))
			return nil
		} else if style == "mt" {
			p.titleDesc.title = append(p.titleDesc.title, strings.TrimSpace(string(text)))
			return nil
		}
		if p.skipUntil == "" {
			if !usfmStyle.Keep {
				if usfmStyle.StyleType != "para" {
					p.skipUntil = style + styleNum + `*`
				}
			}
		} else { // p.skipUntil != ""
			if p.skipUntil == fullStyle.String() {
				p.skipUntil = ""
			}
		}
		whole := strings.TrimSpace(string(text))
		if usfmStyle.StyleType == "para" {
			p.usfmStyle = usfmStyle.StyleType + "." + style + styleNum
			if usfmStyle.Keep && len(whole) > 0 {
				//p.text = append(p.text, whole)
				p.text = append(p.text, string(text))
			}
		} else {
			if p.skipUntil == "" && len(whole) > 0 {
				//p.text = append(p.text, whole)
				p.text = append(p.text, string(text))
			}
		}
	}
	return nil
}

func (p *USFMParser) addChapterHeading(records []db.Script, titles titleDesc) []db.Script {
	var results = make([]db.Script, 0, len(records)+300)
	if len(records) == 0 {
		return results
	}
	var rec = records[0]
	rec.VerseStr = `0`
	rec.VerseNum = 0
	rec.UsfmStyle = `para.mt`
	rec.ScriptTexts = []string{strings.Join(titles.title, " ")}
	results = append(results, rec)
	var lastChapter = 1
	for _, rec = range records {
		if lastChapter != rec.ChapterNum {
			lastChapter = rec.ChapterNum
			var rec2 = rec
			rec2.VerseStr = `0`
			rec2.VerseNum = 0
			rec2.UsfmStyle = `para.h`
			rec2.ScriptTexts = []string{titles.heading + " " + strconv.Itoa(rec.ChapterNum)}
			results = append(results, rec2)
		}
		results = append(results, rec)
	}
	return results
}

/*
func (p *USFMParser) decode(filename string, bookId string) ([]db.Script, titleDesc, *log.Status) {
	var titles titleDesc
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, titles, log.Error(p.ctx, 500, err, "failed to read USFM file")
	}
	styleMap := p.BuildUSFMMap()
	var scripts []db.Script
	var rec = db.Script{DatasetId: 1, BookId: bookId, ChapterNum: 1}
	var chapterNum = 1
	var verseNum int
	var verseStr = "0"
	var skipping = false
	var skipUntil string
	var pendingStyle string

	re := regexp.MustCompile(`\\[a-zA-Z]+\d?\*?`)
	body := string(content)
	indices := re.FindAllStringIndex(body, -1)

	for i, loc := range indices {
		marker := strings.TrimSpace(body[loc[0]+1 : loc[1]])
		textEnd := len(body)
		if i+1 < len(indices) {
			textEnd = indices[i+1][0]
		}
		text := strings.TrimLeft(body[loc[1]:textEnd], " \t\n\r")
		//text := body[loc[1]:textEnd]
		// handle closing marker
		if strings.HasSuffix(marker, "*") {
			if skipping && marker == skipUntil {
				skipping = false
				skipUntil = ""
			}
			continue
		}
		// skip content until closing marker is found
		if skipping {
			continue
		}
		// look up the marker in the style map; strip trailing digit for numbered variants (e.g. \q1 -> q)
		lookupKey := marker
		if len(lookupKey) > 0 {
			last := lookupKey[len(lookupKey)-1]
			if last >= '0' && last <= '9' {
				lookupKey = lookupKey[:len(lookupKey)-1]
			}
		}
		style, found := styleMap[lookupKey]
		if !found {
			if text != "" {
				rec.ScriptTexts = append(rec.ScriptTexts, text)
			}
			continue
		}
		// handle structural types regardless of Keep value
		switch style.StyleType {
		case "book":
			continue
		case "chapter":
			if rec.VerseStr != "" && rec.VerseStr != "0" {
				scripts = append(scripts, rec)
				rec = db.Script{DatasetId: 1, BookId: bookId}
			}
			if fields := strings.Fields(text); len(fields) > 0 {
				if num, err := strconv.Atoi(fields[0]); err == nil {
					chapterNum = num
				}
			}
			verseNum = 0
			verseStr = "0"
			continue
		case "verse":
			if rec.VerseStr != "" && rec.VerseStr != "0" {
				scripts = append(scripts, rec)
				rec = db.Script{DatasetId: 1, BookId: bookId}
			}
			fields := strings.Fields(text)
			if len(fields) > 0 {
				verseStr = fields[0]
				if num, err := strconv.Atoi(verseStr); err == nil {
					verseNum = num
				}
			}
			rec = db.Script{
				DatasetId:  1,
				BookId:     bookId,
				ChapterNum: chapterNum,
				VerseNum:   verseNum,
				VerseStr:   verseStr,
				UsfmStyle:  pendingStyle,
			}
			pendingStyle = ""
			if len(fields) > 1 {
				rec.ScriptTexts = append(rec.ScriptTexts, strings.Join(fields[1:], " "))
			}
			continue
		}
		// if Keep is false, determine whether to skip or just drop
		if !style.Keep {
			if p.hasTerminator(style.StyleType) {
				skipping = true
				skipUntil = marker + `*`
			}
			continue
		}
		// Keep is true — h and mt populate titleDesc rather than the script slice
		if style.StyleType == "para" && lookupKey == "h" {
			titles.heading = text
			continue
		}
		if style.StyleType == "para" && lookupKey == "mt" {
			if text != "" {
				titles.title = append(titles.title, text)
			}
			continue
		}
		if style.StyleType == "para" {
			pendingStyle = style.StyleType + "." + lookupKey
		}
		if text != "" {
			rec.ScriptTexts = append(rec.ScriptTexts, text)
		}
	}
	// save final pending record
	if rec.VerseStr != "" && rec.VerseStr != "0" {
		scripts = append(scripts, rec)
	}
	return scripts, titles, nil
}
*/

/*
// not needed if decode works
func (p *USFMParser) hasTerminator(style string) bool {
	switch style {
	case "book":
		return false
	case "chapter":
		return false
	case "verse":
		return false
	case "para":
		return false
	default:
		return true
	}
}
*/
