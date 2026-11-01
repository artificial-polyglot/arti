package validation

import (
	"context"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/diff"
)

type summaryRow struct {
	Status string
	Verses int
	Chars  int
	CER    string
}

type knownRow struct {
	Reference string
	Status    string
	MinFA     string
	Pctl      string
	CharMin   string
	CharPctl  string
	GeoMin    string
	GeoPctl   string
	CER       string
	Largest   int
	Found     bool
	Missed    bool
	Segments  []segment // arti's diff of the verse text and the ASR
	KnownSegs []segment // the recording team's script line from _Audio_Errors.csv
}

// segment is one piece of the {-}/{+}/{=} diff text: a colored marker followed by plain text.
type segment struct {
	Class  string
	Marker string
	Text   string
}

type reportData struct {
	Dataset    string
	Generated  string
	Cutoff     int
	Summary    []summaryRow
	Validation summaryRow
	Known      []knownRow
	Missed     int
	Total      int
}

func generateReport(ctx context.Context, datasetName string, charCntMap map[PruneSource]diff.Counts, verseCntMap map[PruneSource]int, validation validationSet, knownErrors []knownError) (string, *log.Status) {
	filePath := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), datasetName+"_Validation.html")
	data := reportData{
		Dataset:   datasetName,
		Generated: time.Now().Format("Jan 2, 2006 3:04 pm MST"),
		Cutoff:    CompareRptCutoff,
		Total:     len(knownErrors),
	}
	for _, reas := range []PruneSource{NotPruned, FAScore, Silence, Both} {
		cnt := charCntMap[reas]
		data.Summary = append(data.Summary, summaryRow{
			Status: reas.String(),
			Verses: verseCntMap[reas],
			Chars:  cnt.RefLen,
			CER:    strconv.FormatFloat(cnt.ErrorRate(), 'f', 4, 64),
		})
	}
	data.Validation = summaryRow{
		Status: "All Pruned (validation)",
		Verses: len(validation.verses),
		Chars:  validation.cer.RefLen,
		CER:    strconv.FormatFloat(validation.cer.ErrorRate(), 'f', 4, 64),
	}
	for _, kn := range knownErrors {
		row := knownRow{Reference: kn.reference, Found: kn.found, Largest: kn.largestDiff}
		row.Missed = kn.largestDiff <= CompareRptCutoff
		if row.Missed {
			data.Missed++
		}
		if kn.found {
			row.Status = kn.pruned.String()
			if kn.faRank.minFAScore.Valid {
				row.MinFA = strconv.FormatFloat(kn.faRank.minFAScore.Float64, 'f', 4, 64)
			}
			if kn.faRankFound {
				row.Pctl = strconv.FormatFloat(kn.faRank.percentile, 'f', 1, 64)
			}
			if kn.faRank.charMin.Valid {
				row.CharMin = strconv.FormatFloat(kn.faRank.charMin.Float64, 'f', 6, 64)
				row.CharPctl = strconv.FormatFloat(kn.faRank.charPercentile, 'f', 1, 64)
			}
			if kn.faRank.geoMin.Valid {
				row.GeoMin = strconv.FormatFloat(kn.faRank.geoMin.Float64, 'g', 3, 64)
				row.GeoPctl = strconv.FormatFloat(kn.faRank.geoPercentile, 'f', 1, 64)
			}
			row.CER = strconv.FormatFloat(kn.cer.ErrorRate(), 'f', 3, 64)
			row.Segments = diffSegments(kn.diffs)
		}
		row.KnownSegs = knownSegments(kn.knownText)
		data.Known = append(data.Known, row)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not create validation report at", filePath)
	}
	defer file.Close()
	err = reportTemplate.Execute(file, data)
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not write validation report", filePath)
	}
	err = file.Close()
	if err != nil {
		return filePath, log.Error(ctx, 500, err, "Could not close validation report", filePath)
	}
	return filePath, nil
}

// diffSegments produces the same markers as diff.ReadableDiff, split so each
// marker can be colored: {-} red, {+} green, {=} pale blue.
func diffSegments(diffs []diff.Diff) []segment {
	var result []segment
	for _, df := range diffs {
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

// knownSegments splits an _Audio_Errors.csv text at its {=} (recorded) and {-}
// (missing from the audio) markers, using the same classes as diffSegments.
func knownSegments(text string) []segment {
	var result []segment
	class, marker := "eq", ""
	for {
		next := len(text)
		for _, mk := range []string{"{=}", "{-}"} {
			if i := strings.Index(text, mk); i >= 0 && i < next {
				next = i
			}
		}
		if next > 0 || marker != "" {
			result = append(result, segment{Class: class, Marker: marker, Text: text[:next]})
		}
		if next == len(text) {
			break
		}
		marker = text[next : next+3]
		if marker == "{-}" {
			class = "del"
		} else {
			class = "eq"
		}
		text = text[next+3:]
	}
	return result
}

var reportTemplate = template.Must(template.New("validation").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Dataset}} Validation</title>
<style>
  :root { --text: #1f2328; --muted: #656d76; --border: #d0d7de; --head: #f6f8fa;
          --miss-bg: #fff4f4; --ok: #1a7f37; --bad: #cf222e; }
  body { margin: 0; padding: 24px 16px; background: #ffffff; color: var(--text);
         font: 15px/1.45 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Noto Sans", sans-serif; }
  main { max-width: 1200px; margin: 0 auto; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 17px; margin: 32px 0 8px; }
  .meta { color: var(--muted); font-size: 13px; }
  .result { font-size: 20px; font-weight: 600; margin: 20px 0 0; }
  .result.ok { color: var(--ok); }
  .result.bad { color: var(--bad); }
  .scroll { overflow-x: auto; }
  table { border-collapse: collapse; }
  th, td { border: 1px solid var(--border); padding: 6px 10px; vertical-align: top; text-align: left; }
  th { background: var(--head); font-weight: 600; white-space: nowrap; }
  td.num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
  td.ref { white-space: nowrap; font-weight: 600; }
  td.status { white-space: nowrap; }
  tr.total td { font-weight: 600; border-top: 2px solid var(--muted); }
  tr.missed td { background: var(--miss-bg); }
  tr.missed td.largest { color: var(--bad); font-weight: 600; }
  td.diff { white-space: pre-wrap; word-break: break-word; min-width: 320px; }
  .mk { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-weight: 700;
        border-radius: 3px; padding: 0 2px; }
  .mk.del { background: #ffcfcf; color: #a40e26; }
  .mk.ins { background: #c9f0d1; color: #116329; }
  .mk.eq  { background: #dbeafe; color: #3b6aa0; }
  .notfound { color: var(--bad); font-style: italic; }
  .line + .line { margin-top: 6px; padding-top: 6px; border-top: 1px dashed var(--border); }
  .lbl { display: inline-block; min-width: 44px; color: var(--muted); font-size: 11px;
         text-transform: uppercase; letter-spacing: .03em; }
</style>
</head>
<body>
<main>
<h1>Validation Report: {{.Dataset}}</h1>
<div class="meta">Generated {{.Generated}}. A known error is caught when its largest difference is more than {{.Cutoff}} characters.</div>
<p class="result">Validation CER {{.Validation.CER}}</p>
<div class="meta">The union of all pruned verses: {{.Validation.Verses}} verses, {{.Validation.Chars}} chars.</div>
<p class="result {{if eq .Missed 0}}ok{{else}}bad{{end}}">Missed {{.Missed}} of {{.Total}} known errors</p>

<h2>Character Error Rate by Prune Status</h2>
<div class="scroll">
<table>
<thead><tr><th>Prune Status</th><th>Verses</th><th>Chars</th><th>CER</th></tr></thead>
<tbody>
{{- range .Summary}}
<tr><td class="status">{{.Status}}</td><td class="num">{{.Verses}}</td><td class="num">{{.Chars}}</td><td class="num">{{.CER}}</td></tr>
{{- end}}
</tbody>
<tfoot><tr class="total"><td class="status">{{.Validation.Status}}</td><td class="num">{{.Validation.Verses}}</td><td class="num">{{.Validation.Chars}}</td><td class="num">{{.Validation.CER}}</td></tr></tfoot>
</table>
</div>

<h2>Known Errors</h2>
<div class="scroll">
<table>
<thead><tr><th>Reference</th><th>Prune Status</th><th>Min FA</th><th>FA Pctl</th><th>Char Min</th><th>Char Pctl</th><th title="Minimum over the verse's words of the geometric mean of each word's char fa_score">Geo Min</th><th>Geo Pctl</th><th>CER</th><th>Largest</th><th>Diff</th></tr></thead>
<tbody>
{{- range .Known}}
<tr{{if .Missed}} class="missed"{{end}}>
<td class="ref">{{.Reference}}</td>
{{- if .Found}}
<td class="status">{{.Status}}</td><td class="num">{{.MinFA}}</td><td class="num">{{.Pctl}}</td><td class="num">{{.CharMin}}</td><td class="num">{{.CharPctl}}</td><td class="num">{{.GeoMin}}</td><td class="num">{{.GeoPctl}}</td><td class="num">{{.CER}}</td><td class="num largest">{{.Largest}}</td>
<td class="diff">
<div class="line"><span class="lbl">arti</span>{{range .Segments}}<span class="mk {{.Class}}">{{.Marker}}</span>{{.Text}}{{end}}</div>
<div class="line"><span class="lbl">known</span>{{range .KnownSegs}}<span class="mk {{.Class}}">{{.Marker}}</span>{{.Text}}{{end}}</div></td>
{{- else}}
<td colspan="9"></td><td class="diff">
<div class="line notfound">This verse was not found in data.</div>
<div class="line"><span class="lbl">known</span>{{range .KnownSegs}}<span class="mk {{.Class}}">{{.Marker}}</span>{{.Text}}{{end}}</div>
</td>
{{- end}}
</tr>
{{- end}}
</tbody>
</table>
</div>
</main>
</body>
</html>
`))
