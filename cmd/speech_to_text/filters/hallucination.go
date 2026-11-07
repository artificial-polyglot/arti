package filters

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
)

// A hallucination filter for ASR transcripts.  When ASR produces whole words that are not in the
// text, it checks whether the audio has room for them, using the forced alignment of the text
// done by qa_align.  If a reader really said extra words, forced alignment had to fit the text
// around them: it either leaves a gap between two words, or stretches the words nearby.  Either
// way, that part of the text takes longer than expected.  An insert with no such spare time is
// treated as a hallucination and removed from the transcript.
//
// Spare time is measured in two places, because a diff cannot always say where an insert is:
// when a reader repeats words, the diff may put the insert on either copy.
//   - near the insert: the most spare time of any one word, in a window that extends as many
//     words on each side as the insert has words.
//   - in the whole verse: the spare time of all its words added up.
// An insert is kept when either one has at least NeedFraction of the time the inserted words
// need, and more spare time than NormalPercentile of the verses in the dataset have, so that
// ordinary variation in reading speed does not keep a hallucination.

const MinInsertLetters = 3 // shorter inserts are not judged, and always kept
const NeedFraction = 0.5   // the fraction of the time an insert needs that must be found
const NormalPercentile = 90
const ReplaceWindow = 3    // words of the text on each side of an insert, beyond its own length, to look for a replaced word
const MaxReplaceDiff = 0.5 // the fraction of its chars by which a replaced word may differ

// Clip is the audio given to ASR and qa_align for one verse.
type Clip struct {
	BeginTS float64
	EndTS   float64
}

// Stats describe how the dataset is read.  They are computed from the dataset itself.
type Stats struct {
	SecPerChar    float64 // reading speed, including the spaces between words
	WordGapSec    float64 // median silence between two words
	NearLimitSec  float64 // NormalPercentile of the most spare time of one word in a verse
	VerseLimitSec float64 // NormalPercentile of the spare time of a whole verse
}

// Insert is one run of whole words in a transcript that are not in the text.
type Insert struct {
	ScriptId  int64
	LineRef   generic.VerseRef
	WordIndex int    // the insert comes before this word of the text; len(Words) means at the end
	Text      string // the inserted words
	NeedSec   float64
	NearSec   float64 // spare time near the insert
	VerseSec  float64 // spare time in the whole verse
	Replaces  string  // a nearby word of the text, missing from the transcript, that is like an inserted word
	Kept      bool
}

func (in Insert) Reason() string {
	if in.Replaces != "" {
		return "replacement: like [" + in.Replaces + "] of the text, which the transcript does not have"
	}
	if in.Kept {
		return "audio has " + strconv.FormatFloat(max(in.NearSec, in.VerseSec), 'f', 2, 64) +
			"s spare for " + strconv.FormatFloat(in.NeedSec, 'f', 2, 64) + "s of words"
	}
	return "hallucination: no spare audio for " + strconv.FormatFloat(in.NeedSec, 'f', 2, 64) + "s of words"
}

// FilterHallucinations returns a copy of verses with hallucinated words removed from each
// ASRText, and every whole-word insert with the decision made about it.  verses must have qa_align
// word times, as proofing_rpt.SelectCharData returns them.
func FilterHallucinations(conn db.DBAdapter, verses []generic.Verse2) ([]generic.Verse2, []Insert, *log.Status) {
	clips, status := SelectClips(conn)
	if status != nil {
		return verses, nil, status
	}
	stats := ComputeStats(verses, clips)
	result, inserts := Filter(verses, clips, stats)
	return result, inserts, nil
}

// SelectClips reads the audio that ASR and qa_align were given for each verse.  scripts_qa_align
// begin_ts and end_ts cannot be used, because they are the times of the first and last chars.
func SelectClips(conn db.DBAdapter) (map[int64]Clip, *log.Status) {
	clips := make(map[int64]Clip)
	query := `SELECT script_id, script_begin_ts, script_end_ts FROM scripts`
	rows, err := conn.DB.Query(query)
	if err != nil {
		return clips, log.Error(conn.Ctx, 500, err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var scriptId int64
		var clip Clip
		err = rows.Scan(&scriptId, &clip.BeginTS, &clip.EndTS)
		if err != nil {
			return clips, log.Error(conn.Ctx, 500, err, query)
		}
		clips[scriptId] = clip
	}
	err = rows.Err()
	if err != nil {
		return clips, log.Error(conn.Ctx, 500, err, query)
	}
	return clips, nil
}

func ComputeStats(verses []generic.Verse2, clips map[int64]Clip) Stats {
	var stats Stats
	var gaps []float64
	for _, vs := range verses {
		if len(vs.Words) < 4 {
			continue
		}
		for i := 1; i < len(vs.Words); i++ {
			gaps = append(gaps, vs.Words[i].BeginTS-vs.Words[i-1].EndTS)
		}
	}
	stats.SecPerChar = readingSpeed(verses)
	stats.WordGapSec = percentile(gaps, 50)
	var nearest, totals []float64
	for _, vs := range verses {
		if len(vs.Words) == 0 {
			continue
		}
		spare := spareTime(vs, clips[vs.ScriptId], stats)
		nearest = append(nearest, slices.Max(spare))
		totals = append(totals, sum(spare))
	}
	stats.NearLimitSec = percentile(nearest, NormalPercentile)
	stats.VerseLimitSec = percentile(totals, NormalPercentile)
	return stats
}

// Filter decides about every whole-word insert of every verse, and returns the verses with the
// hallucinated words removed from ASRText.
func Filter(verses []generic.Verse2, clips map[int64]Clip, stats Stats) ([]generic.Verse2, []Insert) {
	result := make([]generic.Verse2, len(verses))
	var inserts []Insert
	for v, vs := range verses {
		result[v] = vs
		if len(vs.Words) == 0 {
			continue
		}
		text := make([]string, len(vs.Words))
		for i, wd := range vs.Words {
			text[i] = strings.ToLower(wd.Text)
		}
		asr := strings.Fields(vs.ASRText)
		spare := spareTime(vs, clips[vs.ScriptId], stats)
		verseSec := sum(spare)
		remove := make([]bool, len(asr))
		removed := false
		runs, matched := wordInserts(text, asr)
		for _, run := range runs {
			words := asr[run.asrBegin:run.asrEnd]
			insText := strings.Join(words, " ")
			if utf8.RuneCountInString(strings.Join(words, "")) < MinInsertLetters {
				continue
			}
			in := Insert{ScriptId: vs.ScriptId, LineRef: vs.LineRef, WordIndex: run.textIndex, Text: insText,
				VerseSec: verseSec}
			in.Replaces = replacedWord(words, text, matched, run.textIndex)
			in.NeedSec = float64(utf8.RuneCountInString(insText)+1) * stats.SecPerChar
			lo := max(0, run.textIndex-len(words))
			hi := min(len(spare)-1, run.textIndex+len(words))
			in.NearSec = slices.Max(spare[lo : hi+1])
			need := NeedFraction * in.NeedSec
			in.Kept = in.Replaces != "" || (in.NearSec >= need && in.NearSec > stats.NearLimitSec) ||
				(in.VerseSec >= need && in.VerseSec > stats.VerseLimitSec)
			if !in.Kept {
				for j := run.asrBegin; j < run.asrEnd; j++ {
					remove[j] = true
				}
				removed = true
			}
			inserts = append(inserts, in)
		}
		if removed {
			var kept []string
			for j, wd := range asr {
				if !remove[j] {
					kept = append(kept, wd)
				}
			}
			result[v].ASRText = strings.Join(kept, " ")
		}
	}
	return result, inserts
}

// spareTime returns, for each word of the verse, the time from the end of the previous word (or
// the start of the clip) to the end of this word, minus the time expected for it.  A last element
// is the time after the last word to the end of the clip, minus the clip's padding.
func spareTime(vs generic.Verse2, clip Clip, stats Stats) []float64 {
	words := vs.Words
	result := make([]float64, 0, len(words)+1)
	hasClip := clip.EndTS > clip.BeginTS && clip.BeginTS <= words[0].BeginTS
	prev, lead := words[0].BeginTS, 0.0
	if hasClip {
		prev, lead = clip.BeginTS, generic.AudioPadSec
	}
	for i, wd := range words {
		expected := float64(utf8.RuneCountInString(wd.Text)) * stats.SecPerChar
		if i == 0 {
			expected += lead
		} else {
			expected += stats.WordGapSec
		}
		result = append(result, (wd.EndTS-prev)-expected)
		prev = wd.EndTS
	}
	tail := 0.0
	if hasClip && clip.EndTS >= prev {
		tail = (clip.EndTS - prev) - generic.AudioPadSec
	}
	return append(result, tail)
}

type insertRun struct {
	textIndex int // the insert comes before this word of the text
	asrBegin  int
	asrEnd    int
}

// wordInserts aligns the words of the text and the transcript by their longest common
// subsequence, and returns each run of transcript words that falls between two matched words
// with no text word between them, and which words of the text are matched.  A run with text
// words on both sides is a replacement, not an insert.
func wordInserts(text []string, asr []string) ([]insertRun, []bool) {
	n, m := len(text), len(asr)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if text[i] == asr[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var runs []insertRun
	matched := make([]bool, n)
	i, j := 0, 0
	prevI, prevJ := 0, 0
	addGap := func(ti int, tj int) {
		if ti == prevI && tj > prevJ {
			runs = append(runs, insertRun{textIndex: ti, asrBegin: prevJ, asrEnd: tj})
		}
	}
	for i < n && j < m {
		if text[i] == asr[j] {
			addGap(i, j)
			matched[i] = true
			i++
			j++
			prevI, prevJ = i, j
		} else if lcs[i+1][j] >= lcs[i][j+1] {
			i++
		} else {
			j++
		}
	}
	addGap(n, m)
	return runs, matched
}

// replacedWord returns a word of the text, missing from the transcript and within ReplaceWindow
// words of an insert, that is the same as an inserted word or differs from it by less than
// MaxReplaceDiff of its chars.  The insert is then a word read in another order, or a misspelled
// word that the alignment matched to another copy of a word nearby, and not a hallucination.
// It returns "" when there is none.
func replacedWord(words []string, text []string, matched []bool, textIndex int) string {
	lo := max(0, textIndex-len(words)-ReplaceWindow)
	hi := min(len(text), textIndex+len(words)+ReplaceWindow)
	for t := lo; t < hi; t++ {
		if matched[t] {
			continue
		}
		for _, wd := range words {
			if similarWords(wd, text[t]) {
				return text[t]
			}
		}
	}
	return ""
}

func similarWords(a string, b string) bool {
	if a == b {
		return true
	}
	cnt := diff.Count(diff.CharLevenshtein(a, b))
	longest := max(utf8.RuneCountInString(a), utf8.RuneCountInString(b))
	return float64(cnt.ErrorCount()) < MaxReplaceDiff*float64(longest)
}

// readingSpeed is the median seconds per char of verses of at least 4 words, counting the spaces.
func readingSpeed(verses []generic.Verse2) float64 {
	var rates []float64
	for _, vs := range verses {
		if len(vs.Words) < 4 {
			continue
		}
		first, last := vs.Words[0], vs.Words[len(vs.Words)-1]
		rates = append(rates, (last.EndTS-first.BeginTS)/float64(numChars(vs.Words)))
	}
	return percentile(rates, 50)
}

func numChars(words []generic.Word2) int {
	count := len(words) - 1 // spaces
	for _, wd := range words {
		count += utf8.RuneCountInString(wd.Text)
	}
	return max(count, 1)
}

func sum(xs []float64) float64 {
	var total float64
	for _, x := range xs {
		total += x
	}
	return total
}

// percentile uses the nearest rank.  It returns 0 for no values.
func percentile(xs []float64, pct float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sorted := slices.Clone(xs)
	slices.Sort(sorted)
	rank := int(math.Ceil(pct/100*float64(len(sorted)))) - 1
	return sorted[min(max(rank, 0), len(sorted)-1)]
}
