package filters

import (
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// A filter for verse boundaries put in the wrong place by forced alignment in mms_align.  When the
// boundary between two verses is too early, the audio of the end of one verse is in the clip of the
// next.  ASR of the first verse is then missing its last letters, and ASR of the next verse begins
// with those letters.  When the boundary is too late, the ASR of the first verse ends with the first
// letters of the next verse, which are missing from the next verse's ASR.
//
// Only a match on both sides is corrected: missing letters in one verse that are found extra in
// the other.  That is what separates a boundary error from a reader who really repeated or left out
// words.  The correction moves the ASR text to the verse it belongs to, moves the play boundary
// between the two verses by the time the moved letters take to read, and spreads the letters of
// the text over that time.  The database is not changed: it keeps what the models produced.
//
// The comparison is by letters, not words, because the boundary is often inside a word, and ASR
// often joins the moved letters to the next word.  Run this filter before FilterHallucinations, so
// that moved words are not judged as inserts.

const MinMoveLetters = 2   // a single letter can match by chance, and is not moved
const MinAnchorLetters = 3 // an equal run of the diff with fewer letters does not end a verse's edge

// BoundaryError is one correction between a verse and the next verse in the same audio file.
type BoundaryError struct {
	ScriptId      int64
	LineRef       generic.VerseRef
	NextScriptId  int64
	NextLineRef   generic.VerseRef
	Text          string // the ASR text that was moved
	Later         bool   // the boundary moved later: Text moved from the start of the next verse to this one
	OldBoundaryTS float64
	NewBoundaryTS float64
}

func (b BoundaryError) Reason() string {
	ts := func(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }
	if b.Later {
		return "[" + b.Text + "] moved from the start of " + refString(b.NextLineRef) + " to the end of " +
			refString(b.LineRef) + ", boundary " + ts(b.OldBoundaryTS) + " -> " + ts(b.NewBoundaryTS)
	}
	return "[" + b.Text + "] moved from the end of " + refString(b.LineRef) + " to the start of " +
		refString(b.NextLineRef) + ", boundary " + ts(b.OldBoundaryTS) + " -> " + ts(b.NewBoundaryTS)
}

// FilterAlignmentErrors returns a copy of verses with the ASR text, play times and char times of
// each boundary error corrected, and every correction made.  verses must have qa_align word times
// and ASR transcripts, as proofing_rpt.SelectCharData and AddASRTranscript return them.
func FilterAlignmentErrors(conn db.DBAdapter, verses []generic.Verse2) ([]generic.Verse2, []BoundaryError, *log.Status) {
	next, status := SelectNextScripts(conn)
	if status != nil {
		return verses, nil, status
	}
	result, errors := FixBoundaries(verses, next, readingSpeed(verses))
	return result, errors, nil
}

// SelectNextScripts returns, for each script, the script that follows it in the same audio file.
func SelectNextScripts(conn db.DBAdapter) (map[int64]int64, *log.Status) {
	next := make(map[int64]int64)
	query := `SELECT script_id, audio_file FROM scripts ORDER BY audio_file, script_begin_ts, script_id`
	rows, err := conn.DB.Query(query)
	if err != nil {
		return next, log.Error(conn.Ctx, 500, err, query)
	}
	defer rows.Close()
	var prevId int64
	var prevFile string
	for rows.Next() {
		var scriptId int64
		var audioFile string
		err = rows.Scan(&scriptId, &audioFile)
		if err != nil {
			return next, log.Error(conn.Ctx, 500, err, query)
		}
		if prevFile == audioFile && prevId != 0 {
			next[prevId] = scriptId
		}
		prevId, prevFile = scriptId, audioFile
	}
	err = rows.Err()
	if err != nil {
		return next, log.Error(conn.Ctx, 500, err, query)
	}
	return next, nil
}

// FixBoundaries corrects each pair of verses that follow each other in verses and in their audio
// file.  secPerChar is the reading speed, used to estimate the time of the moved letters.
func FixBoundaries(verses []generic.Verse2, next map[int64]int64, secPerChar float64) ([]generic.Verse2, []BoundaryError) {
	result := slices.Clone(verses)
	var errors []BoundaryError
	for i := 0; i+1 < len(result); i++ {
		vs, nx := &result[i], &result[i+1]
		if next[vs.ScriptId] != nx.ScriptId || len(vs.Words) == 0 || len(nx.Words) == 0 {
			continue
		}
		_, vsTail, ok1 := edges(vs.Text(), vs.ASRText)
		nxHead, _, ok2 := edges(nx.Text(), nx.ASRText)
		if !ok1 || !ok2 {
			continue
		}
		later := overlap(letters(vsTail.ref), letters(nxHead.asr))
		earlier := overlap(letters(vsTail.asr), letters(nxHead.ref))
		k := max(later, earlier)
		if k < MinMoveLetters {
			continue
		}
		be := BoundaryError{ScriptId: vs.ScriptId, LineRef: vs.LineRef, NextScriptId: nx.ScriptId,
			NextLineRef: nx.LineRef, Later: later >= earlier}
		vsBegin, vsEnd := vs.PlayTimes()
		nxBegin, nxEnd := nx.PlayTimes()
		shift := float64(k) * secPerChar
		be.OldBoundaryTS = vsEnd
		*vs, *nx = copyWords(*vs), copyWords(*nx)
		if be.Later {
			be.Text, nx.ASRText = cutLetters(nx.ASRText, k, true)
			sep := spaceIn(vsTail.lead) || spaceIn(vsTail.ref[:letterIndex(vsTail.ref, len(letters(vsTail.ref))-k)])
			vs.ASRText = join(vs.ASRText, be.Text, sep)
			shift = min(shift, nxEnd-nxBegin)
			be.NewBoundaryTS = vsEnd + shift
			spreadChars(vs.Words, k, false, nxBegin, nxBegin+shift)
		} else {
			be.Text, vs.ASRText = cutLetters(vs.ASRText, k, false)
			sep := spaceIn(nxHead.lead) || spaceIn(nxHead.ref[letterIndex(nxHead.ref, k-1)+1:])
			nx.ASRText = join(be.Text, nx.ASRText, sep)
			shift = min(shift, vsEnd-vsBegin)
			be.NewBoundaryTS = vsEnd - shift
			spreadChars(nx.Words, k, true, vsEnd-shift, vsEnd)
			shift = -shift
		}
		vs.PlayBeginTS, vs.PlayEndTS = vsBegin, vsEnd+shift
		nx.PlayBeginTS, nx.PlayEndTS = nxBegin+shift, nxEnd
		errors = append(errors, be)
	}
	return result, errors
}

// edge is the text at the start or end of a verse, up to the first equal run of the diff with at
// least MinAnchorLetters: ref holds the text's side of it, and asr the transcript's.  lead holds
// the anchor's spaces next to the edge.
type edge struct {
	ref  []rune
	asr  []rune
	lead []rune
}

func edges(ref string, asr string) (head edge, tail edge, ok bool) {
	diffs := diff.DiffMatchPatch(ref, asr)
	first, last := -1, -1
	for i, df := range diffs {
		if df.Type == diffmatchpatch.DiffEqual && len(letters([]rune(df.Text))) >= MinAnchorLetters {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return head, tail, false
	}
	add := func(e *edge, df diffmatchpatch.Diff) {
		text := []rune(df.Text)
		if df.Type != diffmatchpatch.DiffInsert {
			e.ref = append(e.ref, text...)
		}
		if df.Type != diffmatchpatch.DiffDelete {
			e.asr = append(e.asr, text...)
		}
	}
	for _, df := range diffs[:first] {
		add(&head, df)
	}
	for _, df := range diffs[last+1:] {
		add(&tail, df)
	}
	anchor := []rune(diffs[first].Text)
	for i := 0; i < len(anchor) && unicode.IsSpace(anchor[i]); i++ {
		head.lead = append(head.lead, anchor[i])
	}
	anchor = []rune(diffs[last].Text)
	for i := len(anchor) - 1; i >= 0 && unicode.IsSpace(anchor[i]); i-- {
		tail.lead = append(tail.lead, anchor[i])
	}
	return head, tail, true
}

// overlap returns the most letters at the end of a that are the letters at the start of b.
func overlap(a []rune, b []rune) int {
	for k := min(len(a), len(b)); k > 0; k-- {
		if slices.Equal(a[len(a)-k:], b[:k]) {
			return k
		}
	}
	return 0
}

func letters(text []rune) []rune {
	var result []rune
	for _, r := range text {
		if !unicode.IsSpace(r) {
			result = append(result, r)
		}
	}
	return result
}

// letterIndex returns the index in text of its letter number n, or len(text) when there is none.
func letterIndex(text []rune, n int) int {
	count := 0
	for i, r := range text {
		if !unicode.IsSpace(r) {
			if count == n {
				return i
			}
			count++
		}
	}
	return len(text)
}

func spaceIn(text []rune) bool {
	return slices.ContainsFunc(text, unicode.IsSpace)
}

// cutLetters removes k letters, and the spaces among them, from the start (or the end) of text.
// It returns the letters removed and the text that is left.
func cutLetters(text string, k int, fromStart bool) (string, string) {
	runes := []rune(strings.TrimSpace(text))
	if fromStart {
		i := letterIndex(runes, k)
		return strings.TrimSpace(string(runes[:i])), strings.TrimSpace(string(runes[i:]))
	}
	n := len(letters(runes))
	i := letterIndex(runes, n-k)
	return strings.TrimSpace(string(runes[i:])), strings.TrimSpace(string(runes[:i]))
}

func join(a string, b string, space bool) string {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if space && a != "" && b != "" {
		return a + " " + b
	}
	return a + b
}

// copyWords gives a verse its own copy of its words and chars, so they can be changed.
func copyWords(vs generic.Verse2) generic.Verse2 {
	vs.Words = slices.Clone(vs.Words)
	for i := range vs.Words {
		vs.Words[i].Chars = slices.Clone(vs.Words[i].Chars)
	}
	return vs
}

// spreadChars gives the first (or last) k chars of words even times from beginTS to endTS, and
// updates the times of the words they are in.
func spreadChars(words []generic.Word2, k int, fromStart bool, beginTS float64, endTS float64) {
	type pos struct{ w, c int }
	var chars []pos
	for w := range words {
		for c := range words[w].Chars {
			chars = append(chars, pos{w, c})
		}
	}
	k = min(k, len(chars))
	if k == 0 {
		return
	}
	if fromStart {
		chars = chars[:k]
	} else {
		chars = chars[len(chars)-k:]
	}
	step := (endTS - beginTS) / float64(k)
	for i, p := range chars {
		ch := &words[p.w].Chars[p.c]
		ch.BeginTS = beginTS + float64(i)*step
		ch.EndTS = beginTS + float64(i+1)*step
	}
	for w := chars[0].w; w <= chars[len(chars)-1].w; w++ {
		wd := &words[w]
		if len(wd.Chars) > 0 {
			wd.BeginTS = wd.Chars[0].BeginTS
			wd.EndTS = wd.Chars[len(wd.Chars)-1].EndTS
		}
	}
}

func refString(ref generic.VerseRef) string {
	return ref.BookId + " " + strconv.Itoa(ref.ChapterNum) + ":" + ref.VerseStr
}
