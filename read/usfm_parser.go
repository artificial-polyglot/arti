package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

type USFMParser struct {
	ctx  context.Context
	conn db.DBAdapter
}

func NewUSFMParser(conn db.DBAdapter) USFMParser {
	var p USFMParser
	p.ctx = conn.Ctx
	p.conn = conn
	return p
}

func (p *USFMParser) ProcessFiles(inputFiles []input.InputFile) *log.Status {
	var status *log.Status
	for _, file := range inputFiles {
		filename := filepath.Join(file.Directory, file.Filename)
		var records []db.Script
		var titles titleDesc
		records, titles, status = p.decode(filename, file.BookId) // Also edits out non-script elements
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
	var titles titleDesc
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, titles, log.Error(p.ctx, 500, err, "failed to read USFM file")
	}
	styleMap := p.BuildUSFMMap()
	var scripts []db.Script
	var rec db.Script
	var chapterNum = 1
	var verseNum int
	var verseStr = "0"
	var skipping = false
	var skipUntil string

	re := regexp.MustCompile(`\\[a-zA-Z]+\d?\*?`)
	body := string(content)
	indices := re.FindAllStringIndex(body, -1)

	for i, loc := range indices {
		marker := strings.TrimSpace(body[loc[0]+1 : loc[1]])
		textEnd := len(body)
		if i+1 < len(indices) {
			textEnd = indices[i+1][0]
		}
		text := strings.TrimSpace(body[loc[1]:textEnd])
		fmt.Println(loc[0], loc[1], body[loc[0]:loc[1]], marker, text)
		// handle closing markers
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
		// look up the marker in the style map
		style, found := styleMap[marker]
		fmt.Println("style", style.StyleType, style.Keep, found)
		if !found {
			if text != "" {
				rec.ScriptTexts = append(rec.ScriptTexts, text)
			}
			fmt.Println("*** no style", style.StyleType, style.Keep)
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
		// Keep is true — handle by StyleType
		fmt.Println("-- AT SWITCH", style.StyleType)
		switch style.StyleType {
		case "book":
			continue
		case "chapter":
			if len(rec.ScriptTexts) > 0 {
				fmt.Println("*** chapter", rec)
				scripts = append(scripts, rec)
				rec = db.Script{}
			}
			num, err := strconv.Atoi(strings.Fields(text)[0])
			if err == nil {
				chapterNum = num
			}
			verseNum = 0
			verseStr = "0"
		case "verse":
			if len(rec.ScriptTexts) > 0 {
				fmt.Println("*** verse", rec)
				scripts = append(scripts, rec)
				rec = db.Script{}
			}
			verseStr = strings.Fields(text)[0]
			num, err := strconv.Atoi(verseStr)
			if err == nil {
				verseNum = num
			}
			rec = db.Script{
				DatasetId:  1,
				BookId:     bookId,
				ChapterNum: chapterNum,
				VerseNum:   verseNum,
				VerseStr:   verseStr,
			}
			fmt.Println(rec)
			fields := strings.Fields(text)
			if len(fields) > 1 {
				rec.ScriptTexts = append(rec.ScriptTexts, strings.Join(fields[1:], " "))
			}
		default:
			if rec.UsfmStyle == "" {
				rec.UsfmStyle = marker
			}
			if text != "" {
				rec.ScriptTexts = append(rec.ScriptTexts, text)
			}
		}
	}
	// save final pending record
	if len(rec.ScriptTexts) > 0 {
		scripts = append(scripts, rec)
	}
	return scripts, titles, nil
}

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
