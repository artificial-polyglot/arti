package filters

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/diff"
)

// RunFilters runs, in series, the filters that settings turn on, and keeps what each one did, for
// WriteReport.  verses must have qa_align chars and ASR transcripts, as proofing_rpt.SelectCharData
// and AddASRTranscript return them.

// Results are the records of every filter run, and the verses before and after each one.
type Results struct {
	Settings    request.STTFilters
	PairRate    float64
	SpaceHyphen []SpaceHyphenChange
	Boundaries  []BoundaryError
	Pairs       map[rune]map[rune]CharPair // the common pairs found
	CharPairs   []CharPairChange
	Stats       Stats // of the hallucination filter
	Inserts     []Insert
	Missing     []MissingWord
	stages      []stage
	index       map[int64]int // script_id to the index of the verse
}

// stage is the verses after one filter, and the char error counts of each verse.
type stage struct {
	name   string
	verses []generic.Verse2
	counts []diff.Counts
}

const (
	stageBefore         = "Before filters"
	stageSpaceHyphen    = "Spaces and hyphens"
	stageBoundaries     = "Alignment boundaries"
	stageCharPairs      = "Char pairs"
	stageHallucinations = "Hallucinations"
	stageMissingWords   = "Missing words"
)

func RunFilters(conn db.DBAdapter, verses []generic.Verse2, settings request.STTFilters) ([]generic.Verse2, Results, *log.Status) {
	res := Results{Settings: settings, PairRate: DefaultPairRate, index: make(map[int64]int)}
	for i, vs := range verses {
		res.index[vs.ScriptId] = i
	}
	res.addStage(stageBefore, verses)
	var status *log.Status
	if settings.SpaceHyphen {
		verses, res.SpaceHyphen = FilterSpaceHyphen(verses)
		res.addStage(stageSpaceHyphen, verses)
	}
	if settings.FABoundary {
		verses, res.Boundaries, status = FilterAlignmentErrors(conn, verses)
		if status != nil {
			return verses, res, status
		}
		res.addStage(stageBoundaries, verses)
	}
	if settings.CharPairs {
		res.Pairs = FindCharPairs(verses, res.PairRate)
		verses, res.CharPairs = FilterCharPairs(verses, res.PairRate)
		res.addStage(stageCharPairs, verses)
	}
	if settings.Hallucinations {
		clips, status := SelectClips(conn)
		if status != nil {
			return verses, res, status
		}
		res.Stats = ComputeStats(verses, clips)
		verses, res.Inserts = Filter(verses, clips, res.Stats)
		res.addStage(stageHallucinations, verses)
	}
	if settings.MissingWords {
		verses, res.Missing, status = FilterMissingWords(conn, verses)
		if status != nil {
			return verses, res, status
		}
		res.addStage(stageMissingWords, verses)
	}
	return verses, res, nil
}

// AnyOn returns true if settings turn on any filter.
func AnyOn(settings request.STTFilters) bool {
	return settings.SpaceHyphen || settings.FABoundary || settings.CharPairs || settings.Hallucinations ||
		settings.MissingWords
}

// addStage keeps the verses after a filter.  The error counts are computed again only for the
// verses whose transcript the filter changed.
func (r *Results) addStage(name string, verses []generic.Verse2) {
	st := stage{name: name, verses: verses, counts: make([]diff.Counts, len(verses))}
	var prior *stage
	if len(r.stages) > 0 {
		prior = &r.stages[len(r.stages)-1]
	}
	for i, vs := range verses {
		if prior != nil && prior.verses[i].ASRText == vs.ASRText {
			st.counts[i] = prior.counts[i]
		} else if len(vs.Words) > 0 {
			st.counts[i] = diff.Count(diff.CharLevenshtein(vs.Text(), vs.ASRText))
		}
	}
	r.stages = append(r.stages, st)
}

// stageIndex returns the index of the stage of a filter, or -1 if it was not run.
func (r *Results) stageIndex(name string) int {
	return slices.IndexFunc(r.stages, func(st stage) bool { return st.name == name })
}

// verseLines returns the diff of a verse before and after a filter.
func (r *Results) verseLines(stageName string, scriptId int64, label string) []line {
	s := r.stageIndex(stageName)
	i, ok := r.index[scriptId]
	if s < 1 || !ok {
		return nil
	}
	before, after := r.stages[s-1].verses[i], r.stages[s].verses[i]
	if label != "" {
		label += " "
	}
	if before.ASRText == after.ASRText {
		return []line{{Label: label + "diff", Segs: verseSegments(after)}}
	}
	return []line{{Label: label + "before", Segs: verseSegments(before)},
		{Label: label + "after", Segs: verseSegments(after)}}
}

// --- the report ---

type summaryRow struct {
	Stage   string
	Records string
	Outcome string
	Verses  int // verses whose transcript the filter changed
	Errors  int
	CER     string
	Run     bool
}

type section struct {
	Id     string
	Title  string
	Intro  string
	Notes  []string
	Tables []table
}

type table struct {
	Title   string
	Headers []string
	Rows    []row
	Diff    bool // the table has a diff column
	Open    bool
}

type row struct {
	Class   string
	Cells   []cell // the cells before the changes
	Changes []change
	Post    []cell // the cells after the changes
	Lines   []line
}

type cell struct {
	Text  string
	Class string
}

type change struct {
	From string // the transcript's version
	To   string // the version the filter gave it
	Note string
}

type line struct {
	Label string
	Segs  []segment
}

// segment is one piece of the {-}/{+}/{=} diff text: a colored marker followed by plain text.
type segment struct {
	Class  string
	Marker string
	Text   string
}

type reportData struct {
	Dataset   string
	Generated string
	BeforeCER string
	AfterCER  string
	Better    bool
	Summary   []summaryRow
	Sections  []section
}

const openRows = 200 // a table with more rows starts closed

// WriteReport writes an html report of every filter of a run to FCBH_DATASET_TMP.
func WriteReport(ctx context.Context, datasetName string, res Results) (string, *log.Status) {
	filePath := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), datasetName+"_Filters.html")
	data := res.reportData(datasetName)
	file, err := os.Create(filePath)
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not create filter report at", filePath)
	}
	defer file.Close()
	err = filterTemplate.Execute(file, data)
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not write filter report", filePath)
	}
	err = file.Close()
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not close filter report", filePath)
	}
	return filePath, nil
}

func (r *Results) reportData(datasetName string) reportData {
	data := reportData{Dataset: datasetName, Generated: time.Now().Format("Jan 2, 2006 3:04 pm MST")}
	type filterInfo struct {
		name    string
		run     bool
		records int
		outcome string
	}
	removed, confirmed := 0, 0
	for _, in := range r.Inserts {
		if !in.Kept {
			removed++
		}
	}
	for _, m := range r.Missing {
		if m.Confirmed {
			confirmed++
		}
	}
	infos := []filterInfo{
		{stageSpaceHyphen, r.Settings.SpaceHyphen, len(r.SpaceHyphen), itoa(len(r.SpaceHyphen)) + " corrected"},
		{stageBoundaries, r.Settings.FABoundary, len(r.Boundaries), itoa(len(r.Boundaries)) + " moved"},
		{stageCharPairs, r.Settings.CharPairs, len(r.CharPairs), itoa(len(r.CharPairs)) + " corrected"},
		{stageHallucinations, r.Settings.Hallucinations, len(r.Inserts),
			itoa(removed) + " removed, " + itoa(len(r.Inserts)-removed) + " kept"},
		{stageMissingWords, r.Settings.MissingWords, len(r.Missing),
			itoa(confirmed) + " confirmed, " + itoa(len(r.Missing)-confirmed) + " doubtful"},
	}
	before := r.summaryRow(0)
	before.Stage, before.Run = stageBefore, true
	data.Summary = append(data.Summary, before)
	data.BeforeCER = before.CER
	data.AfterCER = before.CER
	for _, fi := range infos {
		sr := summaryRow{Stage: fi.name, Run: fi.run}
		if s := r.stageIndex(fi.name); s > 0 {
			sr = r.summaryRow(s)
			sr.Stage, sr.Run = fi.name, true
			sr.Records, sr.Outcome = itoa(fi.records), fi.outcome
			data.AfterCER = sr.CER
		}
		data.Summary = append(data.Summary, sr)
	}
	data.Better = data.AfterCER < data.BeforeCER
	if r.Settings.SpaceHyphen {
		data.Sections = append(data.Sections, r.spaceHyphenSection())
	}
	if r.Settings.FABoundary {
		data.Sections = append(data.Sections, r.boundarySection())
	}
	if r.Settings.CharPairs {
		data.Sections = append(data.Sections, r.charPairSection())
	}
	if r.Settings.Hallucinations {
		data.Sections = append(data.Sections, r.hallucinationSection())
	}
	if r.Settings.MissingWords {
		data.Sections = append(data.Sections, r.missingSection())
	}
	return data
}

// summaryRow returns the error counts after a stage, and the verses it changed.
func (r *Results) summaryRow(s int) summaryRow {
	var total diff.Counts
	var sr summaryRow
	st := r.stages[s]
	for i, cnt := range st.counts {
		total.Add(cnt)
		if s > 0 && r.stages[s-1].verses[i].ASRText != st.verses[i].ASRText {
			sr.Verses++
		}
	}
	sr.Errors = total.ErrorCount()
	sr.CER = strconv.FormatFloat(total.ErrorRate(), 'f', 4, 64)
	return sr
}

func (r *Results) spaceHyphenSection() section {
	sec := section{Id: "space-hyphen", Title: stageSpaceHyphen,
		Intro: "Where a part of a transcript differs from the text only by spaces and hyphens, the transcript is given the text's version."}
	var joined, split, hyphen int
	for _, c := range r.SpaceHyphen {
		switch {
		case containsHyphen(c.Text) || containsHyphen(c.ASR):
			hyphen++
		case len(c.Text) > len(c.ASR):
			joined++
		default:
			split++
		}
	}
	sec.Notes = []string{itoa(joined) + " words that ASR joined, " + itoa(split) + " words that ASR split, " +
		itoa(hyphen) + " hyphens."}
	tb := table{Title: "Corrections", Headers: []string{"Reference", "Corrections", "Diff"}, Diff: true}
	r.groupRows(&tb, stageSpaceHyphen, len(r.SpaceHyphen), func(i int) (int64, generic.VerseRef, change) {
		c := r.SpaceHyphen[i]
		return c.ScriptId, c.LineRef, change{From: c.ASR, To: c.Text}
	})
	sec.Tables = append(sec.Tables, tb)
	return sec
}

func (r *Results) boundarySection() section {
	sec := section{Id: "boundaries", Title: stageBoundaries,
		Intro: "Where the text missing at the end of one verse is the extra text at the start of the next, or the reverse, " +
			"the text is moved to its verse and the boundary between them is moved."}
	tb := table{Title: "Moved text", Headers: []string{"Reference", "Moved", "Boundary", "Diff"}, Diff: true}
	for _, b := range r.Boundaries {
		var note string
		if b.Later {
			note = "from the start of " + refString(b.NextLineRef) + " to the end of " + refString(b.LineRef)
		} else {
			note = "from the end of " + refString(b.LineRef) + " to the start of " + refString(b.NextLineRef)
		}
		rw := row{Cells: []cell{{Text: refString(b.LineRef) + " / " + refString(b.NextLineRef), Class: "ref"}},
			Changes: []change{{To: b.Text, Note: note}}}
		rw.Post = []cell{{Text: fmtSec(b.OldBoundaryTS) + " → " + fmtSec(b.NewBoundaryTS), Class: "num"}}
		rw.Lines = append(r.verseLines(stageBoundaries, b.ScriptId, refString(b.LineRef)),
			r.verseLines(stageBoundaries, b.NextScriptId, refString(b.NextLineRef))...)
		tb.Rows = append(tb.Rows, rw)
	}
	tb.Open = len(tb.Rows) <= openRows
	sec.Tables = append(sec.Tables, tb)
	return sec
}

func (r *Results) charPairSection() section {
	sec := section{Id: "char-pairs", Title: stageCharPairs,
		Intro: "A pair of chars is common when it replaces more than " + strconv.FormatFloat(r.PairRate, 'f', 1, 64) +
			" percent of the occurrences of its char.  Diacritics, equivalent vowels and ignorable chars are undone at any frequency.  " +
			"A part of a transcript is corrected only when every difference in it is explained."}
	ruleCounts := make(map[string]int)
	pairCounts := make(map[[2]rune]int)
	for _, c := range r.CharPairs {
		for _, rule := range c.Rules {
			ruleCounts[rule]++
		}
		for _, p := range c.Pairs {
			pairCounts[[2]rune{p.From, p.To}]++
		}
	}
	pairs := table{Title: "Common pairs", Headers: []string{"Text", "ASR", "Replaced", "Occurrences", "Rate", "Corrected"}, Open: true}
	var found []CharPair
	for _, tos := range r.Pairs {
		for _, p := range tos {
			found = append(found, p)
		}
	}
	slices.SortFunc(found, func(a, b CharPair) int { return b.Count - a.Count })
	for _, p := range found {
		pairs.Rows = append(pairs.Rows, row{Cells: []cell{
			{Text: string(p.From) + " " + codePoint(p.From), Class: "char"},
			{Text: string(p.To) + " " + codePoint(p.To), Class: "char"},
			{Text: itoa(p.Count), Class: "num"}, {Text: itoa(p.CharCount), Class: "num"},
			{Text: strconv.FormatFloat(p.Rate, 'f', 1, 64) + "%", Class: "num"},
			{Text: itoa(pairCounts[[2]rune{p.From, p.To}]), Class: "num"}}})
	}
	if len(found) == 0 {
		sec.Notes = append(sec.Notes, "No pair is common enough.")
	} else {
		sec.Tables = append(sec.Tables, pairs)
	}
	rules := table{Title: "Rules", Headers: []string{"Rule", "Corrected"}, Open: true}
	for _, rule := range []string{RuleDiacritic, RuleVowel, RuleIgnorable} {
		rules.Rows = append(rules.Rows, row{Cells: []cell{{Text: rule}, {Text: itoa(ruleCounts[rule]), Class: "num"}}})
	}
	sec.Tables = append(sec.Tables, rules)
	tb := table{Title: "Corrections", Headers: []string{"Reference", "Corrections", "Diff"}, Diff: true}
	r.groupRows(&tb, stageCharPairs, len(r.CharPairs), func(i int) (int64, generic.VerseRef, change) {
		c := r.CharPairs[i]
		notes := slices.Clone(c.Rules)
		for _, p := range c.Pairs {
			notes = append(notes, "pair "+p.String())
		}
		return c.ScriptId, c.LineRef, change{From: c.ASR, To: c.Text, Note: strings.Join(notes, ", ")}
	})
	sec.Tables = append(sec.Tables, tb)
	return sec
}

func (r *Results) hallucinationSection() section {
	sec := section{Id: "hallucinations", Title: stageHallucinations,
		Intro: "Each run of whole words that ASR wrote and the text does not have is removed, unless the audio has spare time for it: " +
			"at least " + strconv.FormatFloat(NeedFraction*100, 'f', 0, 64) + " percent of the time the words need, near the words or in the whole verse."}
	sec.Notes = []string{"Reading speed " + strconv.FormatFloat(r.Stats.SecPerChar, 'f', 3, 64) + " s/char, word gap " +
		fmtSec(r.Stats.WordGapSec) + ", normal spare time near a word " + fmtSec(r.Stats.NearLimitSec) +
		" and in a verse " + fmtSec(r.Stats.VerseLimitSec) + " (the " + itoa(NormalPercentile) + "th percentile)."}
	headers := []string{"Reference", "Words", "Need", "Near", "Verse", "Diff"}
	removed := table{Title: "Removed", Headers: headers, Diff: true}
	kept := table{Title: "Kept", Headers: headers, Diff: true}
	for _, in := range r.Inserts {
		rw := row{Cells: []cell{{Text: refString(in.LineRef), Class: "ref"}},
			Changes: []change{{From: in.Text}}}
		rw.Post = []cell{{Text: fmtSec(in.NeedSec), Class: "num"}, {Text: fmtSec(in.NearSec), Class: "num"},
			{Text: fmtSec(in.VerseSec), Class: "num"}}
		rw.Lines = r.verseLines(stageHallucinations, in.ScriptId, "")
		if in.Kept {
			rw.Changes[0] = change{To: in.Text}
			kept.Rows = append(kept.Rows, rw)
		} else {
			removed.Rows = append(removed.Rows, rw)
		}
	}
	removed.Open = len(removed.Rows) <= openRows
	kept.Open = len(kept.Rows) <= openRows
	sec.Tables = append(sec.Tables, removed, kept)
	return sec
}

func (r *Results) missingSection() section {
	sec := section{Id: "missing-words", Title: stageMissingWords,
		Intro: "Each whole word of the text that is missing from the transcript is checked with the mms_align score of the base model.  " +
			"It is confirmed missing from the audio when the score is below " + strconv.FormatFloat(MMSLimit, 'f', 2, 64) +
			".  Transcripts are not changed.  Numbers are not checked."}
	headers := []string{"Reference", "Word", "QA Score", "MMS Score", "Diff"}
	doubtful := table{Title: "Doubtful: the base model may have heard the word", Headers: headers, Diff: true}
	confirmed := table{Title: "Confirmed missing from the audio", Headers: headers, Diff: true}
	for _, m := range r.Missing {
		rw := row{Cells: []cell{{Text: refString(m.LineRef), Class: "ref"}, {Text: m.Text, Class: "word"},
			{Text: strconv.FormatFloat(m.QAScore, 'f', 3, 64), Class: "num"},
			{Text: strconv.FormatFloat(m.MMSScore, 'f', 3, 64), Class: "num"}}}
		rw.Lines = r.verseLines(stageMissingWords, m.ScriptId, "")
		if m.Confirmed {
			confirmed.Rows = append(confirmed.Rows, rw)
		} else {
			rw.Class = "missed"
			doubtful.Rows = append(doubtful.Rows, rw)
		}
	}
	doubtful.Open = len(doubtful.Rows) <= openRows
	confirmed.Open = len(confirmed.Rows) <= openRows
	sec.Tables = append(sec.Tables, doubtful, confirmed)
	return sec
}

// groupRows adds a row for each verse with changes, which are in verse order.
func (r *Results) groupRows(tb *table, stageName string, n int, get func(int) (int64, generic.VerseRef, change)) {
	var prior int64 = -1
	for i := range n {
		scriptId, ref, ch := get(i)
		if scriptId != prior || len(tb.Rows) == 0 {
			tb.Rows = append(tb.Rows, row{Cells: []cell{{Text: refString(ref), Class: "ref"}},
				Lines: r.verseLines(stageName, scriptId, "")})
			prior = scriptId
		}
		last := &tb.Rows[len(tb.Rows)-1]
		last.Changes = append(last.Changes, ch)
	}
	tb.Open = len(tb.Rows) <= openRows
}

// verseSegments produces the same markers as diff.ReadableDiff, split so each marker can be
// colored: {-} red, {+} green, {=} pale blue.
func verseSegments(vs generic.Verse2) []segment {
	var result []segment
	for _, df := range diff.DiffReplace(vs.Text(), vs.ASRText) {
		switch df.Type {
		case diff.OpEqual:
			result = append(result, segment{Class: "eq", Marker: "{=}", Text: df.Text})
		case diff.OpInsert:
			result = append(result, segment{Class: "ins", Marker: "{+}", Text: df.Text})
		case diff.OpDelete:
			result = append(result, segment{Class: "del", Marker: "{-}", Text: df.Text})
		case diff.OpReplace:
			result = append(result, segment{Class: "del", Marker: "{-r}", Text: df.Text})
			result = append(result, segment{Class: "ins", Marker: "{+r}", Text: df.Replace})
		}
	}
	return result
}

func containsHyphen(s string) bool {
	for _, r := range s {
		if isHyphen(r) {
			return true
		}
	}
	return false
}

func codePoint(r rune) string {
	return fmt.Sprintf("U+%04X", r)
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func fmtSec(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64) + "s"
}

var filterTemplate = template.Must(template.New("filters").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Dataset}} Filters</title>
<style>
  :root { --text: #1f2328; --muted: #656d76; --border: #d0d7de; --head: #f6f8fa;
          --miss-bg: #fff4f4; --ok: #1a7f37; --bad: #cf222e; }
  body { margin: 0; padding: 24px 16px; background: #ffffff; color: var(--text);
         font: 15px/1.45 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Noto Sans", sans-serif; }
  main { max-width: 1200px; margin: 0 auto; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 17px; margin: 36px 0 6px; }
  h3 { font-size: 15px; margin: 18px 0 6px; }
  p { margin: 6px 0; }
  .meta { color: var(--muted); font-size: 13px; }
  .result { font-size: 20px; font-weight: 600; margin: 20px 0 0; }
  .result.ok { color: var(--ok); }
  .result.bad { color: var(--bad); }
  nav a { margin-right: 14px; font-size: 13px; }
  .scroll { overflow-x: auto; }
  table { border-collapse: collapse; }
  th, td { border: 1px solid var(--border); padding: 6px 10px; vertical-align: top; text-align: left; }
  th { background: var(--head); font-weight: 600; white-space: nowrap; }
  td.num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
  td.ref { white-space: nowrap; font-weight: 600; }
  td.outcome { white-space: nowrap; }
  td.word, td.char { white-space: nowrap; }
  tr.notrun td { color: var(--muted); }
  tr.total td { font-weight: 600; border-top: 2px solid var(--muted); }
  tr.missed td { background: var(--miss-bg); }
  td.diff { white-space: pre-wrap; word-break: break-word; min-width: 320px; }
  td.changes { min-width: 180px; }
  .chg + .chg { margin-top: 4px; }
  .from { background: #ffcfcf; color: #a40e26; border-radius: 3px; padding: 0 3px; white-space: pre-wrap; }
  .to   { background: #c9f0d1; color: #116329; border-radius: 3px; padding: 0 3px; white-space: pre-wrap; }
  .note { color: var(--muted); font-size: 12px; }
  .mk { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-weight: 700;
        border-radius: 3px; padding: 0 2px; }
  .mk.del { background: #ffcfcf; color: #a40e26; }
  .mk.ins { background: #c9f0d1; color: #116329; }
  .mk.eq  { background: #dbeafe; color: #3b6aa0; }
  .line + .line { margin-top: 6px; padding-top: 6px; border-top: 1px dashed var(--border); }
  .lbl { display: inline-block; min-width: 52px; color: var(--muted); font-size: 11px;
         text-transform: uppercase; letter-spacing: .03em; }
  details > summary { cursor: pointer; font-weight: 600; margin: 14px 0 6px; }
  details > summary .meta { font-weight: 400; }
</style>
</head>
<body>
<main>
<h1>Filter Report: {{.Dataset}}</h1>
<div class="meta">Generated {{.Generated}}. The filters run in the order below; each one is given the transcripts the one before it made.</div>
<p class="result {{if .Better}}ok{{end}}">CER {{.BeforeCER}} → {{.AfterCER}}</p>
<div class="meta">The character error rate of all verses, before and after the filters.</div>
<nav>{{range .Sections}}<a href="#{{.Id}}">{{.Title}}</a>{{end}}</nav>

<h2>Summary</h2>
<div class="scroll">
<table>
<thead><tr><th>Filter</th><th>Records</th><th>Outcome</th><th>Verses Changed</th><th>Error Chars</th><th>CER</th></tr></thead>
<tbody>
{{- range .Summary}}
{{- if .Run}}
<tr><td>{{.Stage}}</td><td class="num">{{.Records}}</td><td class="outcome">{{.Outcome}}</td><td class="num">{{if .Records}}{{.Verses}}{{end}}</td><td class="num">{{.Errors}}</td><td class="num">{{.CER}}</td></tr>
{{- else}}
<tr class="notrun"><td>{{.Stage}}</td><td colspan="5">not run</td></tr>
{{- end}}
{{- end}}
</tbody>
</table>
</div>

{{- range .Sections}}
<h2 id="{{.Id}}">{{.Title}}</h2>
<p class="meta">{{.Intro}}</p>
{{- range .Notes}}<p>{{.}}</p>{{end}}
{{- range .Tables}}
<details{{if .Open}} open{{end}}>
<summary>{{.Title}} <span class="meta">({{len .Rows}})</span></summary>
{{- if .Rows}}
<div class="scroll">
<table>
<thead><tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr></thead>
<tbody>
{{- range .Rows}}
<tr{{if .Class}} class="{{.Class}}"{{end}}>
{{- range .Cells}}<td class="{{.Class}}">{{.Text}}</td>{{end}}
{{- if .Changes}}<td class="changes">{{range .Changes}}<div class="chg">{{if .From}}<span class="from">{{.From}}</span>{{end}}{{if and .From .To}} → {{end}}{{if .To}}<span class="to">{{.To}}</span>{{end}}{{if .Note}} <span class="note">{{.Note}}</span>{{end}}</div>{{end}}</td>{{end}}
{{- range .Post}}<td class="{{.Class}}">{{.Text}}</td>{{end}}
{{- if .Lines}}<td class="diff">{{range .Lines}}<div class="line"><span class="lbl">{{.Label}}</span>{{range .Segs}}<span class="mk {{.Class}}">{{.Marker}}</span>{{.Text}}{{end}}</div>{{end}}</td>{{end}}
</tr>
{{- end}}
</tbody>
</table>
</div>
{{- end}}
</details>
{{- end}}
{{- end}}
</main>
</body>
</html>
`))
