package proofing_rpt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/artificial-polyglot/arti/decode_yaml/request"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

// Versions used:
// <link href="https://cdn.datatables.net/v/dt/jq-3.7.0/dt-2.3.8/datatables.min.css" rel="stylesheet">
//<script src="https://cdn.datatables.net/v/dt/jq-3.7.0/dt-2.3.8/datatables.min.js"></script>

type HTMLWriter struct {
	ctx         context.Context
	datasetName string
	s3Client    s3_datastore.S3Client
	cutoff      float64
	out         *os.File
}

func NewHTMLWriter(ctx context.Context, datasetName string) HTMLWriter {
	var h HTMLWriter
	h.ctx = ctx
	h.datasetName = datasetName
	h.s3Client, _ = s3_datastore.NewS3Client(ctx)
	return h
}

func (h *HTMLWriter) WriteReport(records [][]Word, verses map[int64]Verse, baseURL string,
	languageISO string, asr request.SpeechToText) (string, *log.Status) {
	var err error
	var model string
	h.cutoff = computeCutoff(records)
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
	for _, words := range records {
		verse := verses[words[0].ScriptId]
		h.WriteLine(words, verse, baseURL)
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
		<span><input type="number" id="accuracyCutoff" min="0" step="0.01" style="width: 40px;" value="0.01">
		<label for="accuracyCutoff"> Accuracy Cutoff</label></span>
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
		<th>Accuracy</th>
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

func (h *HTMLWriter) WriteLine(words []Word, verse Verse, baseURL string) {
	_, _ = h.out.WriteString("<tr>\n")
	h.writeCell(strconv.FormatInt(words[0].ScriptId, 10))
	h.writeCell(strconv.FormatFloat(ComputeAccuracy(words, h.cutoff), 'f', 3, 64))
	h.writeCell(strconv.FormatFloat(startTime(words), 'f', 3, 64))
	h.writeCell(strconv.FormatFloat(duration(words), 'f', 3, 64))
	var params []string
	params = append(params, "this")
	signedURL := h.s3Client.SignAudioURL(baseURL, verse.AudioFile)
	params = append(params, "'"+signedURL+"'")
	params = append(params, strconv.FormatFloat(words[0].BeginTS, 'f', 4, 64))
	params = append(params, strconv.FormatFloat(findEndTS(words), 'f', 4, 64))
	h.writeCell("<button title=\"" + minSecFormat(words[0].BeginTS) + "\" onclick=\"playVerse(" + strings.Join(params, ",") + ")\">Play</button>")
	h.writeCell(verse.Ref.Description())
	_, _ = h.out.WriteString(`<td>`)
	var span string
	for _, wd := range words {
		if wd.Ttype != "W" {
			span = wd.Word
		} else if wd.Word == wd.URoman && wd.Opacity == 0 {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f>%s</span>`,
				wd.WordId, wd.FaScore, wd.BeginTS, wd.EndTS, wd.Word)
		} else if wd.Word == wd.URoman {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f style="background-color:rgba(255,0,0,%f2);">%s</span>`,
				wd.WordId, wd.FaScore, wd.BeginTS, wd.EndTS, wd.Opacity, wd.Word)
		} else if wd.Opacity == 0 {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f data-word="%s" data-uroman="%s">%s</span>`,
				wd.WordId, wd.FaScore, wd.BeginTS, wd.EndTS, wd.Word, wd.URoman, wd.Word)
		} else {
			span = fmt.Sprintf(`<span id="w-%d" title="%.3f" data-begin=%.3f data-end=%.3f data-word="%s" data-uroman="%s" style="background-color:rgba(255,0,0,%f2);">%s</span>`,
				wd.WordId, wd.FaScore, wd.BeginTS, wd.EndTS, wd.Word, wd.URoman, wd.Opacity, wd.Word)
		}
		_, _ = h.out.WriteString(span)
		//if wd.Ttype == "W" && wd.FaScore < FA_SCORE_CUTOFF {
		//	_, _ = h.out.WriteString(fmt.Sprintf("(%.2f)", wd.FaScore))
		//}
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
                { "orderable": false, "targets": [2,3,4,5,6] }
				// { "visible": false, "targets": [8] }  
            ],
            "pageLength": 50,
            "lengthMenu": [[50, 500, -1], [50, 500, "All"]],
			"order": [[ 1, "desc" ]]
        });
    	$.fn.dataTable.ext.search.push(function(settings, data, dataIndex) {
        	var hideZeros = $('#hideVerse0').prop('checked');
        	if (!hideZeros) return true;
        	return !data[5].endsWith(":0"); 
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
			var cutoff = parseFloat($('#accuracyCutoff').val());
			if (isNaN(cutoff)) return true;          // empty/invalid → show all
			var accuracy = parseFloat(data[1]);      // column 1 = Accuracy
			if (isNaN(accuracy)) return true;        // non-numeric cell → don't filter
			return accuracy >= cutoff;               // keep at/below cutoff, drop above
		});
		table.draw();
		$('#accuracyCutoff').on('change', function() {
			table.draw();
		});
	});

`
	_, _ = h.out.WriteString(script)
	script = `let playbackRate = 1;
$('#playSpeed').on('change', function () {
	playbackRate = parseFloat(this.value);
	if (currentAudio) currentAudio.playbackRate = playbackRate;   // change speed live
});
let currentSpans = null;
let currentAudio = null;
let currentEndTS = null;                 // <-- remember where to stop

function playVerse(button, audioFile, beginTS, endTS) {
    if (currentAudio) {
        currentAudio.pause();
        clearHighlight();
    }
    currentEndTS = endTS;                 // <-- store it for onTimeUpdate

    const scope = button.closest('tr');
    currentSpans = Array.from(scope.querySelectorAll('span[data-begin]')).filter(span => {
        const start = parseFloat(span.dataset.begin);
        return start >= beginTS && start < endTS;
    });

    currentAudio = new Audio(audioFile);
	currentAudio.playbackRate = playbackRate;      // <-- apply the chosen speed
	currentAudio.addEventListener('loadedmetadata', () => {
    	currentAudio.currentTime = beginTS;
	});

    // seek to the verse start once the browser knows the file's duration
    currentAudio.addEventListener('loadedmetadata', () => {
        currentAudio.currentTime = beginTS;
    });
    currentAudio.addEventListener('timeupdate', onTimeUpdate);
    currentAudio.addEventListener('ended', clearHighlight);
    currentAudio.play().catch(err => console.error('play failed', err));
}

function onTimeUpdate() {
    const t = currentAudio.currentTime;

    if (currentEndTS !== null && t >= currentEndTS) {
        currentAudio.pause();
        clearHighlight();
        return;
    }
    clearHighlight();
    // the current word is the LAST one whose begin time has been reached
    const word = currentSpans
        .filter(span => parseFloat(span.dataset.begin) <= t)
        .pop();
    if (word) word.classList.add('highlight');
}

function clearHighlight() {
    if (currentSpans) {
        currentSpans.forEach(span => span.classList.remove('highlight'));
    }
}
	</script>
</body>
</html>
`
	_, _ = h.out.WriteString(script)
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

func findEndTS(words []Word) float64 {
	for i := len(words) - 1; i >= 0; i-- {
		if words[i].Ttype == "W" {
			return words[i].EndTS
		}
	}
	return words[0].EndTS
}

func ComputeAccuracy(words []Word, cutoff float64) float64 {
	var numWords float64
	var numBelow float64
	var minimum = 1.0
	for _, w := range words {
		if w.Ttype == "W" {
			numWords += 1.0
			if w.FaScore < minimum {
				minimum = w.FaScore
			}
			if w.FaScore < cutoff {
				numBelow += 1.0
			}
		}
	}
	pctBelow := numBelow / numWords
	invMin := 1.0 - minimum
	score := pctBelow * invMin
	return score
}

func computeCutoff(words [][]Word) float64 {
	var faScores []float64
	for _, verse := range words {
		for _, w := range verse {
			if w.Ttype == "W" {
				faScores = append(faScores, w.FaScore)
			}
		}
	}
	sort.Float64s(faScores)
	pos := int(float64(len(faScores)) * 0.01)
	cutoff := faScores[pos]
	return cutoff
}

func startTime(words []Word) float64 {
	for _, w := range words {
		if w.Ttype == "W" {
			return w.BeginTS
		}
	}
	return 0.0
}
func duration(words []Word) float64 {
	if len(words) > 0 {
		return findEndTS(words) - startTime(words)
	} else {
		return 0.0
	}
}
