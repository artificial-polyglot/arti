package asr_align

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"unicode/utf8"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/mms"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/utility/ffmpeg"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/utility/performance"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/utility/stdio_exec"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type ASRAlign struct {
	ctx       context.Context
	conn      db.DBAdapter
	lang      string
	sttLang   string
	adapter   bool
	mmsAsrPy  *stdio_exec.StdioExec
	diffMatch *diffmatchpatch.DiffMatchPatch
	//versePattern *regexp.Regexp
	testing bool // set in asr_align_test.go
	timer   performance.CodeTimer
}

func NewASRAlign(ctx context.Context, conn db.DBAdapter, lang string, sttLang string, adapter bool) ASRAlign {
	var a ASRAlign
	a.ctx = ctx
	a.conn = conn
	a.lang = lang
	a.sttLang = sttLang
	a.adapter = adapter
	a.diffMatch = diffmatchpatch.New()
	//a.versePattern = regexp.MustCompile(`\{(\d+)\}`)
	a.timer = performance.NewCodeTimer()
	return a
}

func (a *ASRAlign) ProcessFiles(files []input.InputFile) *log.Status {
	a.timer.Duration("Start")
	var status *log.Status
	tempDir, err := os.MkdirTemp(os.Getenv(`FCBH_DATASET_TMP`), "mms_asr_align_")
	if err != nil {
		return log.Error(a.ctx, 500, err, `Error creating temp dir`)
	}
	defer os.RemoveAll(tempDir)
	var lang = a.lang
	if a.sttLang != "" {
		lang = a.sttLang
	}
	if !a.adapter {
		lang, status = mms.CheckLanguage(a.ctx, a.lang, a.sttLang, "mms_asr")
		if status != nil {
			return status
		}
	}
	status = a.conn.UpdateASRLanguage(lang)
	if status != nil {
		return status
	}
	pythonScript := filepath.Join(os.Getenv("GOPROJ"), "mms/asr_align/asr_align.py")
	var useAdapter string
	if a.adapter {
		useAdapter = "adapter"
	}
	a.mmsAsrPy, status = stdio_exec.NewStdioExec(a.ctx, os.Getenv(`FCBH_MMS_ASR_PYTHON`), pythonScript, lang, useAdapter)
	if status != nil {
		return status
	}
	defer a.mmsAsrPy.Close()
	a.timer.Duration("After Setup")
	for _, file := range files {
		status = a.processFile(file, tempDir)
		if status != nil {
			return status
		}
	}
	return status
}

func (a *ASRAlign) processFile(file input.InputFile, tempDir string) *log.Status {
	response, status := a.processASR(file, tempDir)
	if status != nil {
		return status
	}
	a.timer.Duration("After ASR")
	var audioChars []Char
	err := json.Unmarshal([]byte(response), &audioChars)
	if err != nil {
		return log.Error(a.ctx, 500, err, "Error Unmarshalling ASR Response")
	}
	a.timer.Duration("After Unmarshal")
	textChars, status1 := a.selectVersesByBookChapter(file.BookId, file.Chapter)
	if status1 != nil {
		return status1
	}
	a.timer.Duration("After select book")
	mergeChars := a.merge(audioChars, textChars)
	a.timer.Duration("After Merge")
	DumpChars(textChars)
	DumpChars(audioChars)
	DumpChars(mergeChars)
	//DumpEndTS(mergeChars)
	words := a.summarizeCharsToWords(mergeChars)
	//for _, w := range words {
	//	word := selectWord(a.conn.DB, w.WordId)
	//	fmt.Println(w.WordId, word, w.EndTS)
	//}
	scripts := a.summarizeWordsToScripts(words)
	//checkForZeroScriptId(scripts)
	err = a.updateWords(a.conn.DB, words)
	if err != nil {
		return log.Error(a.ctx, 500, err, "Error updating timestamps in word table")
	}
	err = a.updateScripts(a.conn.DB, scripts)
	if err != nil {
		return log.Error(a.ctx, 500, err, "Error updating timestamps in scripts table")
	}
	for _, s := range scripts {
		text := selectScript(a.conn.DB, s.ScriptId)
		fmt.Println(s.ScriptId, s.EndTS, text)
	}
	return nil
}

func (a *ASRAlign) processASR(file input.InputFile, tempDir string) (string, *log.Status) {
	var response string
	testFile := file.MediaId + "_" + file.BookId + "_" + strconv.Itoa(file.Chapter) + ".txt"
	_, err := os.Stat(testFile)
	if a.testing && !os.IsNotExist(err) {
		bytes, err2 := os.ReadFile(testFile)
		if err2 != nil {
			return response, log.Error(a.ctx, 500, err2, "Error Reading Test File.")
		}
		response = string(bytes)
	} else {
		wavFile, status := ffmpeg.ConvertMp3ToWav(a.ctx, tempDir, file.FilePath())
		if status != nil {
			return response, status
		}
		response, status = a.mmsAsrPy.Process(wavFile)
		if status != nil {
			return response, status
		}
		if a.testing {
			err = os.WriteFile(testFile, []byte(response), 0644)
			if err != nil {
				return response, log.Error(a.ctx, 500, err, "Error Writing Test File.")
			}
		}
	}
	return response, nil
}

type RuneDiff struct {
	Type diffmatchpatch.Operation
	Rune rune
}

/*
*
Compare the two []Char.  Output a []Char that is assigned VerseStr, and include
*/
func (a *ASRAlign) merge(audioChars []Char, textChars []Char) []Char {
	var results []Char
	var audioStr = a.convertChar2String(audioChars)
	var textStr = a.convertChar2String(textChars)
	lenAudio := utf8.RuneCountInString(audioStr)
	lenText := utf8.RuneCountInString(textStr)
	fmt.Println(lenAudio, lenText)
	diffs := a.diffMatch.DiffMain(audioStr, textStr, false)
	var runeDiffs []RuneDiff
	for _, diff := range diffs {
		for _, r := range diff.Text {
			runeDiffs = append(runeDiffs, RuneDiff{Type: diff.Type, Rune: r})
		}
	}
	var audioIndex = 0
	var textIndex = 0
	for _, diff := range runeDiffs {
		var selected Char
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			selected = textChars[textIndex]
			selected.BeginTS = audioChars[audioIndex].BeginTS
			selected.EndTS = audioChars[audioIndex].EndTS
			results = append(results, selected)
			audioIndex++
			textIndex++
		case diffmatchpatch.DiffDelete:
			if audioChars[audioIndex].Char != 32 { // remove space chars that are only in the audio
				selected = audioChars[audioIndex]
				if textIndex < len(textChars) { // not sure why this is needed
					selected.ScriptId = textChars[textIndex].ScriptId
					selected.WordId = textChars[textIndex].WordId
					selected.VerseStr = textChars[textIndex].VerseStr
					selected.WordSeq = textChars[textIndex].WordSeq
				}
				results = append(results, selected)
				//audioIndex++
			}
			audioIndex++
		case diffmatchpatch.DiffInsert:
			selected = textChars[textIndex]
			results = append(results, selected)
			textIndex++
		}
	}
	return results
}

func (a *ASRAlign) convertChar2String(chars []Char) string {
	var result []rune
	for _, ch := range chars {
		result = append(result, ch.Char)
	}
	return string(result)
}

func (a *ASRAlign) mergeAddedWords(chars []Char) []Char {
	// This solution is not good, it requires the entire words table to be recreated with new word_id
	// And we are doing this one chapter at a time

	// This method should iterate over the merged Chars
	// The added words will not have a wordId, or a verseStr
	// The verseId should be assigned by what is before and after them.
	// When there is a different verseStr before and after, I think the before vs should be assigned
	// That is, they should be put at the end of the prior verse, rather than the beginning.
	// The wordId should be assigned as 1 more than the prior word, and all words that
	// are already assigned should be incremented by one
	//
	// Not sure when the update is done.  The inserted words could be put aside and inserted as a group.
	// And the updated words could be also set aside and inserted as a group.
	// The changed words need to be updated before the inserts
	return chars
}

type Script struct {
	ScriptId int64
	BeginTS  float64
	EndTS    float64
}
type Word struct {
	ScriptId int64
	WordId   int64
	Word     string // or space
	BeginTS  float64
	EndTS    float64
}

/*
	func (a *ASRAlign) summarizeCharsToWords(chars []Char) []Word {
		var words []Word
		if len(chars) == 0 {
			return words
		}
		current := Word{
			ScriptId: chars[0].ScriptId,
			WordId:   chars[0].WordId,
			BeginTS:  chars[0].BeginTS,
			EndTS:    chars[0].EndTS,
		}
		for _, c := range chars[1:] {
			if c.WordId != current.WordId {
				words = append(words, current)
				current = Word{
					ScriptId: c.ScriptId,
					WordId:   c.WordId,
					BeginTS:  c.BeginTS,
					EndTS:    c.EndTS,
				}
			}
			current.EndTS = c.EndTS
		}
		return append(words, current)
	}
*/
func (a *ASRAlign) summarizeCharsToWords(chars []Char) []Word {
	var words []Word
	i := 0
	for i < len(chars) {
		c := chars[i]
		if c.Char == 32 {
			words = append(words, Word{
				ScriptId: c.ScriptId,
				WordId:   c.WordId,
				Word:     " ",
				BeginTS:  c.BeginTS,
				EndTS:    c.EndTS,
			})
			i++
		} else {
			// Accumulate non-space chars into a word
			j := i
			for j < len(chars) && chars[j].Char != ' ' {
				j++
			}
			segment := chars[i:j]
			var sb []rune
			for _, ch := range segment {
				sb = append(sb, ch.Char)
			}
			words = append(words, Word{
				ScriptId: segment[0].ScriptId,
				WordId:   segment[0].WordId,
				Word:     string(sb),
				BeginTS:  segment[0].BeginTS,
				EndTS:    segment[len(segment)-1].EndTS,
			})
			i = j
		}
	}
	return words
}

func (a *ASRAlign) summarizeWordsToScripts(words []Word) []Script {
	var scripts []Script
	if len(words) == 0 {
		return scripts
	}
	current := Script{
		ScriptId: words[0].ScriptId,
		BeginTS:  words[0].BeginTS,
		EndTS:    words[0].EndTS,
	}
	for _, w := range words[1:] {
		if w.ScriptId != current.ScriptId {
			scripts = append(scripts, current)
			current = Script{
				ScriptId: w.ScriptId,
				BeginTS:  w.BeginTS,
				EndTS:    w.EndTS,
			}
		}
		current.EndTS = w.EndTS
	}
	return append(scripts, current)
}

// For debug only
func checkForZeroScriptId(chars []Script) {
	fmt.Println("\n\n\n ***** Zero check ****")
	for _, c := range chars {
		if c.ScriptId == 0 {
			fmt.Println("zero scriptId", c)
		}
		//if c.WordId == 0 {
		//	fmt.Println("zero wordId", c)
		//}
	}
}
