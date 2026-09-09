package proofing_rpt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	audioplayer "github.com/artificial-polyglot/arti/cmd/output"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

// Versions used:
// <link href="https://cdn.datatables.net/v/dt/jq-3.7.0/dt-2.3.8/datatables.min.css" rel="stylesheet">
//<script src="https://cdn.datatables.net/v/dt/jq-3.7.0/dt-2.3.8/datatables.min.js"></script>

type HTMLWriter struct {
	ctx         context.Context
	datasetName string
	cutoff      float64
	out         *os.File
}

func NewHTMLWriter(ctx context.Context, datasetName string) HTMLWriter {
	var h HTMLWriter
	h.ctx = ctx
	h.datasetName = datasetName
	return h
}

func (h *HTMLWriter) WriteReport(verses []Verse2, audioURLs map[string]generic.AudioFile,
	languageISO string, asr request.SpeechToText) (string, *log.Status) {
	var err error
	var model string
	switch asr {
	case request.SpeechToText{MMS: true}:
		model = "Model: MMS"
	case request.SpeechToText{MMSAdapter: true}:
		model = "Model: MMS Adapter"
	case request.SpeechToText{Wav2Vec2ASR: true}:
		model = "Model: Wav2Vec2 Word"
	default:
		model = ""
	}
	h.out, err = os.Create(filepath.Join(os.Getenv(`FCBH_DATASET_TMP`), h.datasetName+"_proof.html"))
	if err != nil {
		return "", log.Error(h.ctx, 500, err, `Error creating output file for proof`)
	}
	filename := h.WriteHeading(languageISO, model)
	for _, vs := range verses {
		verse := vs.LineRef
		key := verse.BookId + strconv.Itoa(verse.ChapterNum)
		audioURL := audioURLs[key]
		h.WriteLine(vs, audioURL)
	}
	h.WriteEnd()
	return filename, nil
}

func (h *HTMLWriter) WriteHeading(languageISO string, model string) string {
	head := `<!DOCTYPE html>
<html>
 <head>
  <meta charset="utf-8">
  <title>Audio Proofing Report</title>
`
	_, _ = h.out.WriteString(head)
	_, _ = h.out.WriteString(`<link rel="stylesheet" type="text/css" href="https://cdn.datatables.net/v/dt/jq-3.7.0/dt-2.3.8/datatables.min.css">`)
	_, _ = h.out.WriteString("</head><body>\n")
	_, _ = h.out.WriteString(`<h2 style="text-align:center">Proof `)
	_, _ = h.out.WriteString(h.datasetName)
	_, _ = h.out.WriteString("</h2>\n")
	_, _ = h.out.WriteString(`<h3 style="text-align:center">`)
	_, _ = h.out.WriteString(model)
	_, _ = h.out.WriteString(`   ASR ISO `)
	_, _ = h.out.WriteString(languageISO)
	_, _ = h.out.WriteString(`</h3>`)
	_, _ = h.out.WriteString(`<h3 style="text-align:center">`)
	loc, _ := time.LoadLocation("America/Denver")
	_, _ = h.out.WriteString(time.Now().In(loc).Format(`Mon Jan 2 2006 03:04:05 pm MST`))
	_, _ = h.out.WriteString("</h3>\n")
	controls := `<div style="display: flex; justify-content: space-evenly; align-items: center; margin: 30px; width=90%">
		<span><input type="number" id="scoreCutoff" min="0" step="0.01" style="width: 60px;" value="0.01">
		<label for="scoreCutoff"> Score Cutoff</label></span>
		<span><input type="checkbox" id="hideVerse0" checked><label for="hideVerse0">Hide Headings</label></span>
		<span><input type="checkbox" id="showUroman"><label for="showUroman">Show Uroman</label></span>
		<span><select id="playSpeed">
        	<option value="1">Normal</option>
        	<option value="0.75">Slower (0.75×)</option>
        	<option value="0.5">Slowest (0.5×)</option>
    		</select><label for="playSpeed">Speed</label></span>
	</div>
`
	_, _ = h.out.WriteString(controls)
	_, _ = h.out.WriteString("<audio id='validateAudio'></audio>\n")
	table := `<table id="diffTable" class="display">
    <thead>
    <tr>
        <th>Line</th>
		<th>Score</th>
		<th>Chars</th>
		<th>Start</th>
		<th>Duration</th>
		<th>Button</th>
        <th>Ref</th>
		<th>Source Text</th>
    </tr>
    </thead>
    <tbody>
`
	_, _ = h.out.WriteString(table)
	return h.out.Name()
}

func (h *HTMLWriter) WriteLine(verse Verse2, audioURL generic.AudioFile) {
	_, _ = h.out.WriteString("<tr>\n")
	h.writeCell(strconv.FormatInt(verse.ScriptId, 10))
	h.writeCell(strconv.FormatFloat(ComputeMinimum(verse.Words), 'f', 4, 64))
	_, _ = h.out.WriteString(`<td class="lowScoreChars"></td>`)
	h.writeCell(strconv.FormatFloat(startTime(verse.Words), 'f', 2, 64))
	h.writeCell(strconv.FormatFloat(verse.Duration, 'f', 2, 64))
	var params []string
	params = append(params, "this")
	params = append(params, "'"+audioURL.UnsignedURL+"'")
	params = append(params, strconv.FormatFloat(verse.BeginTS, 'f', 4, 64))
	params = append(params, strconv.FormatFloat(verse.EndTS, 'f', 4, 64))
	h.writeCell("<button title=\"" + minSecFormat(verse.BeginTS) + "\" onclick=\"playVerse(" + strings.Join(params, ",") + ")\">Play</button>")
	h.writeCell(verse.LineRef.Description())
	_, _ = h.out.WriteString(`<td>`)
	var span string
	for _, wd := range verse.Words {
		if wd.IsASR {
			span = wd.Text
		} else if wd.IsASR && wd.Text == wd.Uroman {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f style="background-color:rgb(255, 193, 84);">%s</span>`,
				wd.WordId, wd.FAScore, wd.BeginTS, wd.EndTS, wd.WordPunct)
		} else if wd.IsASR {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f data-word="%s" data-uroman="%s" style="background-color:rgb(255, 193, 84);">%s</span>`,
				wd.WordId, wd.FAScore, wd.BeginTS, wd.EndTS, wd.WordPunct, wd.Uroman, wd.WordPunct)
		} else if wd.Text == wd.Uroman && wd.Opacity == 0 {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f>%s</span>`,
				wd.WordId, wd.FAScore, wd.BeginTS, wd.EndTS, wd.WordPunct)
		} else if wd.Text == wd.Uroman {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f style="background-color:rgba(255,0,0,%f2);">%s</span>`,
				wd.WordId, wd.FAScore, wd.BeginTS, wd.EndTS, wd.Opacity, wd.WordPunct)
		} else if wd.Opacity == 0 {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f data-word="%s" data-uroman="%s">%s</span>`,
				wd.WordId, wd.FAScore, wd.BeginTS, wd.EndTS, wd.WordPunct, wd.Uroman, wd.WordPunct)
		} else {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f data-word="%s" data-uroman="%s" style="background-color:rgba(255,0,0,%f2);">%s</span>`,
				wd.WordId, wd.FAScore, wd.BeginTS, wd.EndTS, wd.WordPunct, wd.Uroman, wd.Opacity, wd.WordPunct)
		}
		_, _ = h.out.WriteString(span)
		_, _ = h.out.WriteString(" ")
	}
	_, _ = h.out.WriteString("</td></tr>\n")
}

func (h *HTMLWriter) writeCell(content string) {
	_, _ = h.out.WriteString(`<td>`)
	_, _ = h.out.WriteString(content)
	_, _ = h.out.WriteString(`</td>`)
}

func (h *HTMLWriter) WriteEnd() {
	table := `</tbody>
	</table>
`
	_, _ = h.out.WriteString(table)
	_, _ = h.out.WriteString(`<script type="text/javascript" src="https://cdn.datatables.net/v/dt/jq-3.7.0/dt-2.3.8/datatables.min.js"></script>`)
	_, _ = h.out.WriteString("\n")
	style := `<style>
	.dataTables_length select {
		width: auto;
		display: inline-block;
		padding: 5px;
		margin-left: 5px;
		border-radius: 4px;
		border: 1px solid #ccc;
	}
	.dataTables_filter input {
		width: auto;
		display: inline-block;
		padding: 5px;
		border-radius: 4px;
		border: 1px solid #ccc;
	}
	.highlight {
    	background-color: #ffe680;   /* soft yellow */
    	border-radius: 3px;
    	padding: 0 1px;              /* tiny breathing room around the word */
	}
	.dataTables_wrapper .dataTables_length, .dataTables_wrapper .dataTables_filter {
		margin-bottom: 20px;
	}
	</style>
`
	_, _ = h.out.WriteString(style)
	script := `<script>
    $(document).ready(function() {
        var table = $('#diffTable').DataTable({
            "columnDefs": [
                { "orderable": false, "targets": [1,3,4,5,6,7] }
				// { "visible": false, "targets": [8] }
            ],
            "pageLength": 50,
            "lengthMenu": [[50, 500, -1], [50, 500, "All"]],
			"order": [[ 2, "desc" ]]
        });
    	$.fn.dataTable.ext.search.push(function(settings, data, dataIndex) {
        	var hideZeros = $('#hideVerse0').prop('checked');
        	if (!hideZeros) return true;
        	return !data[6].endsWith(":0"); 
    	});
		$('#hideVerse0').prop('checked', true);
		table.draw();
		$('#hideVerse0').on('change', function() {
			table.draw();
		});
		function applyUroman() {
			var showU = $('#showUroman').prop('checked');
			document.querySelectorAll('span[data-word]').forEach(function(sp) {
				sp.textContent = showU ? sp.dataset.uroman : sp.dataset.word;
			});
		}
		$('#showUroman').on('change', applyUroman);   // user toggles
		table.on('draw', applyUroman);
		$.fn.dataTable.ext.search.push(function(settings, data, dataIndex) {
			var cutoff = parseFloat($('#scoreCutoff').val());
			if (isNaN(cutoff)) return true;       // empty/invalid → show all
			var score = parseFloat(data[1]);      // column 1 = Minimum Score
			if (isNaN(score)) return true;        // non-numeric cell → don't filter
			return score <= cutoff;               // keep at/below cutoff, drop above
		});
		table.draw();
		function updateLowCounts() {
		  var cutoff = parseFloat($('#scoreCutoff').val());
		  if (isNaN(cutoff)) cutoff = 0;
		  table.rows().every(function () {
			var minScore = parseFloat(this.data()[1]);   // column 1 = Minimum Score
			var sum = 0;
			if (!isNaN(minScore) && minScore <= cutoff) {
			  this.node().querySelectorAll('td:last-child span[data-begin]').forEach(function (sp) {
				var score = parseFloat(sp.title);
				if (!isNaN(score) && score <= cutoff) {
				  sum += (sp.dataset.word || sp.textContent).length;
				}
			  });
			}
			this.cell(this.index(), 2).data(sum);   // sets cell + keeps sort correct
		  });
		  table.draw(false);          // false = stay on current page
		}
    	updateLowCounts();
    	$('#scoreCutoff').on('change', function () {
        	updateLowCounts();      // this already calls table.draw(false)
    	});
	});

`
	_, _ = h.out.WriteString(script)
	script = `window.avPlaybackRate = 1;
$('#playSpeed').on('change', function () {
	window.avPlaybackRate = parseFloat(this.value);
	const audio = document.getElementById('validateAudio');
	if (audio && !audio.paused) audio.playbackRate = window.avPlaybackRate;
});

let currentSpans = null;

window.avOnVerseStart = function (button, beginTS, endTS) {
    const scope = button.closest('tr');
    currentSpans = Array.from(scope.querySelectorAll('span[data-begin]')).filter(span => {
        const start = parseFloat(span.dataset.begin);
        return start >= beginTS && start < endTS;
    });
};

window.avOnTimeUpdate = function (audio, t) {
    clearHighlight();
    const word = currentSpans
        .filter(span => parseFloat(span.dataset.begin) <= t)
        .pop();
    if (word) word.classList.add('highlight');
};

window.avOnVerseStop = function () {
    clearHighlight();
};

function clearHighlight() {
    if (currentSpans) {
        currentSpans.forEach(span => span.classList.remove('highlight'));
    }
}
	</script>
`
	_, _ = h.out.WriteString(script)
	_, _ = h.out.WriteString("<script>\n")
	_, _ = h.out.WriteString(audioplayer.Script)
	_, _ = h.out.WriteString("\n</script>\n")
	_, _ = h.out.WriteString("</body>\n</html>\n")
	_ = h.out.Close()
}

func minSecFormat(duration float64) string {
	if duration > 0.5 {
		duration -= 0.5
	} else {
		duration = 0.0
	}
	mins := int(duration / 60.0)
	secs := duration - float64(mins)*60.0
	var minStr string
	var delim string
	if int(mins) > 0 {
		minStr = strconv.FormatInt(int64(mins), 10)
		delim = ":"
	}
	secStr := strconv.FormatFloat(secs, 'f', 0, 64)
	return minStr + delim + secStr
}

func ComputeMinimum(words []Word2) float64 {
	var minimum = 1.0
	for _, w := range words {
		//if w.Ttype == "W" {
		if w.FAScore < minimum {
			minimum = w.FAScore
		}
		//}
	}
	return minimum
}

func startTime(words []Word2) float64 {
	for _, w := range words {
		//if w.Ttype == "W" {
		return w.BeginTS
		//}
	}
	return 0.0
}
