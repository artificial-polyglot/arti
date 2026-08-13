package validate

import (
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/request"
)

// Marshal produces the YAML encoding of a request.Request. It is not a
// generic marshaller - it is hand-written for exactly this struct tree, so
// that the WASM build doesn't need encoding/json, gopkg.in/yaml.v3, or
// reflect (which all three pull in). Field order, indentation (4 spaces
// per level), omitempty behavior, and scalar quoting are written to match
// gopkg.in/yaml.v3's default output for this struct - see
// generate_yaml_test.go, which checks that byte-for-byte.
func Marshal(req request.Request) ([]byte, error) {
	var w yamlBuilder
	w.boolean(0, "is_new", req.IsNew, false)
	w.str(0, "dataset_name", req.DatasetName, false)
	w.str(0, "username", req.Username, false)
	w.str(0, "bible_id", req.BibleId, false)
	w.str(0, "language_iso", req.LanguageISO, false)
	w.str(0, "alt_language", req.AltLanguage, true)
	w.integer(0, "priority", req.Priority, true)
	w.strSlice(0, "notify_ok", req.NotifyOk, false)
	w.strSlice(0, "notify_err", req.NotifyErr, false)
	// Output.Directory has no omitempty tag of its own, so unlike every
	// other nested struct here, "output" can't be skipped just because its
	// rendered content is empty (Directory would still render as
	// directory: "" whenever any other Output field is set). yaml.v3 skips
	// the whole field based on the struct's zero value instead, so match
	// that explicitly.
	if req.Output != (request.Output{}) {
		w.nested(0, "output", func(sub *yamlBuilder, d int) { appendOutput(sub, d, req.Output) })
	}
	w.nested(0, "testament", func(sub *yamlBuilder, d int) { appendTestament(sub, d, req.Testament) })
	w.nested(0, "database", func(sub *yamlBuilder, d int) { appendDatabase(sub, d, req.Database) })
	w.nested(0, "audio_data", func(sub *yamlBuilder, d int) { appendAudioData(sub, d, req.AudioData) })
	w.nested(0, "text_data", func(sub *yamlBuilder, d int) { appendTextData(sub, d, req.TextData) })
	w.nested(0, "sheet_columns", func(sub *yamlBuilder, d int) { appendSheetColumns(sub, d, req.SheetColumns) })
	w.nested(0, "timestamps", func(sub *yamlBuilder, d int) { appendTimestamps(sub, d, req.Timestamps) })
	w.nested(0, "training", func(sub *yamlBuilder, d int) { appendTraining(sub, d, req.Training) })
	w.nested(0, "speech_to_text", func(sub *yamlBuilder, d int) { appendSpeechToText(sub, d, req.SpeechToText) })
	w.nested(0, "stt_decoder", func(sub *yamlBuilder, d int) { appendSTTDecoder(sub, d, req.STTDecoder) })
	w.nested(0, "detail", func(sub *yamlBuilder, d int) { appendDetail(sub, d, req.Detail) })
	w.nested(0, "audio_encoding", func(sub *yamlBuilder, d int) { appendAudioEncoding(sub, d, req.AudioEncoding) })
	w.nested(0, "text_encoding", func(sub *yamlBuilder, d int) { appendTextEncoding(sub, d, req.TextEncoding) })
	w.nested(0, "audio_proof", func(sub *yamlBuilder, d int) { appendAudioProof(sub, d, req.AudioProof) })
	w.nested(0, "compare", func(sub *yamlBuilder, d int) { appendCompare(sub, d, req.Compare) })
	return []byte(w.sb.String()), nil
}

func appendOutput(w *yamlBuilder, depth int, o request.Output) {
	w.str(depth, "directory", o.Directory, false)
	w.boolean(depth, "csv", o.CSV, true)
	w.boolean(depth, "json", o.JSON, true)
	w.boolean(depth, "sqlite", o.Sqlite, true)
}

func appendTestament(w *yamlBuilder, depth int, t request.Testament) {
	w.boolean(depth, "nt", t.NT, true)
	w.strSlice(depth, "nt_books", t.NTBooks, true)
	w.boolean(depth, "ot", t.OT, true)
	w.strSlice(depth, "ot_books", t.OTBooks, true)
}

func appendDatabase(w *yamlBuilder, depth int, d request.Database) {
	w.str(depth, "aws_s3", d.AWSS3, true)
	w.str(depth, "file", d.File, true)
}

func appendAudioData(w *yamlBuilder, depth int, a request.AudioData) {
	w.nested(depth, "bible_brain", func(sub *yamlBuilder, d int) { appendBibleBrainAudio(sub, d, a.BibleBrain) })
	w.str(depth, "file", a.File, true)
	w.str(depth, "aws_s3", a.AWSS3, true)
	w.str(depth, "post", a.POST, true)
	w.boolean(depth, "no_audio", a.NoAudio, true)
}

func appendBibleBrainAudio(w *yamlBuilder, depth int, b request.BibleBrainAudio) {
	w.boolean(depth, "mp3_64", b.MP3_64, true)
	w.boolean(depth, "mp3_16", b.MP3_16, true)
	w.boolean(depth, "opus", b.OPUS, true)
	w.str(depth, "set_type_code", b.SetTypeCode, true)
}

func appendTextData(w *yamlBuilder, depth int, t request.TextData) {
	w.nested(depth, "bible_brain", func(sub *yamlBuilder, d int) { appendBibleBrainText(sub, d, t.BibleBrain) })
	w.str(depth, "file", t.File, true)
	w.str(depth, "aws_s3", t.AWSS3, true)
	w.str(depth, "post", t.POST, true)
	w.boolean(depth, "no_text", t.NoText, true)
}

func appendBibleBrainText(w *yamlBuilder, depth int, b request.BibleBrainText) {
	w.boolean(depth, "text_usx_edit", b.TextUSXEdit, true)
	w.boolean(depth, "text_plain_edit", b.TextPlainEdit, true)
	w.boolean(depth, "text_plain", b.TextPlain, true)
}

func appendSheetColumns(w *yamlBuilder, depth int, s request.SheetColumns) {
	w.integer(depth, "book_col", s.BookCol, true)
	w.integer(depth, "chapter_col", s.ChapterCol, true)
	w.integer(depth, "verse_col", s.VerseCol, true)
	w.integer(depth, "character_col", s.CharacterCol, true)
	w.integer(depth, "actor_col", s.ActorCol, true)
	w.integer(depth, "line_col", s.LineCol, true)
	w.integer(depth, "text_col", s.TextCol, true)
}

func appendTimestamps(w *yamlBuilder, depth int, t request.Timestamps) {
	w.boolean(depth, "bible_brain", t.BibleBrain, true)
	w.boolean(depth, "aeneas", t.Aeneas, true)
	w.boolean(depth, "ts_bucket", t.TSBucket, true)
	w.boolean(depth, "mms_fa_verse", t.MMSFAVerse, true)
	w.boolean(depth, "mms_align", t.MMSAlign, true)
	w.boolean(depth, "no_timestamps", t.NoTimestamps, true)
}

func appendTraining(w *yamlBuilder, depth int, t request.Training) {
	w.boolean(depth, "redo_training", t.RedoTraining, true)
	w.nested(depth, "mms_adapter", func(sub *yamlBuilder, d int) { appendMMSAdapter(sub, d, t.MMSAdapter) })
	w.nested(depth, "wav2vec2_word", func(sub *yamlBuilder, d int) { appendWav2Vec2(sub, d, t.Wav2Vec2Word) })
	w.boolean(depth, "no_training", t.NoTraining, true)
}

func appendMMSAdapter(w *yamlBuilder, depth int, m request.MMSAdapter) {
	w.integer(depth, "batch_mb", m.BatchMB, true)
	w.integer(depth, "num_epochs", m.NumEpochs, true)
	w.float(depth, "learning_rate", m.LearningRate, true)
	w.float(depth, "warmup_pct", m.WarmupPct, true)
	w.float(depth, "grad_norm_max", m.GradNormMax, true)
}

func appendWav2Vec2(w *yamlBuilder, depth int, v request.Wav2Vec2) {
	w.integer(depth, "batch_mb", v.BatchMB, true)
	w.integer(depth, "num_epochs", v.NumEpochs, true)
	w.float(depth, "learning_rate", v.LearningRate, true)
	w.float(depth, "warmup_pct", v.WarmupPct, true)
	w.float(depth, "grad_norm_max", v.GradNormMax, true)
	w.float(depth, "min_audio_sec", v.MinAudioSec, true)
}

func appendSpeechToText(w *yamlBuilder, depth int, s request.SpeechToText) {
	w.boolean(depth, "mms_asr", s.MMS, true)
	w.boolean(depth, "adapter_asr", s.MMSAdapter, true)
	w.boolean(depth, "wav2vec2_asr", s.Wav2Vec2ASR, true)
	w.nested(depth, "whisper", func(sub *yamlBuilder, d int) { appendWhisper(sub, d, s.Whisper) })
	w.boolean(depth, "mms_asr_align", s.MMSASRAlign, true)
	w.boolean(depth, "no_speech_to_text", s.NoSpeechToText, true)
}

func appendWhisper(w *yamlBuilder, depth int, wh request.Whisper) {
	w.nested(depth, "model", func(sub *yamlBuilder, d int) { appendWhisperModel(sub, d, wh.Model) })
}

func appendWhisperModel(w *yamlBuilder, depth int, m request.WhisperModel) {
	w.boolean(depth, "large", m.Large, true)
	w.boolean(depth, "medium", m.Medium, true)
	w.boolean(depth, "small", m.Small, true)
	w.boolean(depth, "base", m.Base, true)
	w.boolean(depth, "tiny", m.Tiny, true)
}

func appendSTTDecoder(w *yamlBuilder, depth int, s request.STTDecoder) {
	w.boolean(depth, "greedy", s.Greedy, true)
	w.boolean(depth, "simple", s.Simple, true)
	w.boolean(depth, "hotwords", s.Hotwords, true)
	w.boolean(depth, "kenlm", s.Kenlm, true)
	w.boolean(depth, "no_stt_decoder", s.NoSTTDecoder, true)
}

func appendDetail(w *yamlBuilder, depth int, d request.Detail) {
	w.boolean(depth, "lines", d.Lines, true)
	w.boolean(depth, "verses", d.Verses, true)
	w.boolean(depth, "words", d.Words, true)
}

func appendAudioEncoding(w *yamlBuilder, depth int, a request.AudioEncoding) {
	w.boolean(depth, "mfcc", a.MFCC, true)
	w.boolean(depth, "no_encoding", a.NoEncoding, true)
}

func appendTextEncoding(w *yamlBuilder, depth int, t request.TextEncoding) {
	w.boolean(depth, "fast_text", t.FastText, true)
	w.boolean(depth, "no_encoding", t.NoEncoding, true)
}

func appendAudioProof(w *yamlBuilder, depth int, a request.AudioProof) {
	w.boolean(depth, "html_report", a.HTMLReport, true)
}

func appendCompare(w *yamlBuilder, depth int, c request.Compare) {
	w.boolean(depth, "html_report", c.HTMLReport, true)
	w.str(depth, "base_dataset", c.BaseDataset, true)
	w.integer(depth, "gordon_filter", c.GordonFilter, true)
	w.nested(depth, "compare_settings", func(sub *yamlBuilder, d int) { appendCompareSettings(sub, d, c.CompareSettings) })
}

func appendCompareSettings(w *yamlBuilder, depth int, c request.CompareSettings) {
	w.boolean(depth, "lower_case", c.LowerCase, true)
	w.boolean(depth, "remove_prompt_chars", c.RemovePromptChars, true)
	w.boolean(depth, "remove_punctuation", c.RemovePunctuation, true)
	w.nested(depth, "double_quotes", func(sub *yamlBuilder, d int) { appendCompareChoice(sub, d, c.DoubleQuotes) })
	w.nested(depth, "apostrophe", func(sub *yamlBuilder, d int) { appendCompareChoice(sub, d, c.Apostrophe) })
	w.nested(depth, "hyphen", func(sub *yamlBuilder, d int) { appendCompareChoice(sub, d, c.Hyphen) })
	w.nested(depth, "diacritical_marks", func(sub *yamlBuilder, d int) { appendDiacriticalChoice(sub, d, c.DiacriticalMarks) })
}

func appendCompareChoice(w *yamlBuilder, depth int, c request.CompareChoice) {
	w.boolean(depth, "remove", c.Remove, true)
	w.boolean(depth, "normalize", c.Normalize, true)
}

func appendDiacriticalChoice(w *yamlBuilder, depth int, d request.DiacriticalChoice) {
	w.boolean(depth, "remove", d.Remove, true)
	w.boolean(depth, "normalize_nfc", d.NormalizeNFC, true)
	w.boolean(depth, "normalize_nfd", d.NormalizeNFD, true)
	w.boolean(depth, "normalize_nfkc", d.NormalizeNFKC, true)
	w.boolean(depth, "normalize_nfkd", d.NormalizeNFKD, true)
}

// ---- low-level YAML writing -------------------------------------------

type yamlBuilder struct {
	sb strings.Builder
}

func (w *yamlBuilder) pad(depth int) string {
	return strings.Repeat("    ", depth)
}

func (w *yamlBuilder) str(depth int, key, val string, omitempty bool) {
	if omitempty && val == "" {
		return
	}
	w.sb.WriteString(w.pad(depth))
	w.sb.WriteString(key)
	w.sb.WriteString(": ")
	w.sb.WriteString(quoteYAMLString(val))
	w.sb.WriteString("\n")
}

func (w *yamlBuilder) boolean(depth int, key string, val bool, omitempty bool) {
	if omitempty && !val {
		return
	}
	w.sb.WriteString(w.pad(depth))
	w.sb.WriteString(key)
	if val {
		w.sb.WriteString(": true\n")
	} else {
		w.sb.WriteString(": false\n")
	}
}

func (w *yamlBuilder) integer(depth int, key string, val int, omitempty bool) {
	if omitempty && val == 0 {
		return
	}
	w.sb.WriteString(w.pad(depth))
	w.sb.WriteString(key)
	w.sb.WriteString(": ")
	w.sb.WriteString(strconv.Itoa(val))
	w.sb.WriteString("\n")
}

func (w *yamlBuilder) float(depth int, key string, val float64, omitempty bool) {
	if omitempty && val == 0 {
		return
	}
	w.sb.WriteString(w.pad(depth))
	w.sb.WriteString(key)
	w.sb.WriteString(": ")
	w.sb.WriteString(strconv.FormatFloat(val, 'g', -1, 64))
	w.sb.WriteString("\n")
}

func (w *yamlBuilder) strSlice(depth int, key string, vals []string, omitempty bool) {
	if omitempty && len(vals) == 0 {
		return
	}
	w.sb.WriteString(w.pad(depth))
	w.sb.WriteString(key)
	w.sb.WriteString(":")
	if len(vals) == 0 {
		w.sb.WriteString(" []\n")
		return
	}
	w.sb.WriteString("\n")
	for _, v := range vals {
		w.sb.WriteString(w.pad(depth + 1))
		w.sb.WriteString("- ")
		w.sb.WriteString(quoteYAMLString(v))
		w.sb.WriteString("\n")
	}
}

// nested appends a struct-valued field as a "key:\n" header followed by its
// indented content, but only if that content is non-empty - every nested
// struct field in Request is tagged omitempty, so an all-zero-value struct
// (e.g. Output{}) must produce no output at all, matching yaml.v3.
func (w *yamlBuilder) nested(depth int, key string, appendFields func(sub *yamlBuilder, depth int)) {
	var sub yamlBuilder
	appendFields(&sub, depth+1)
	if sub.sb.Len() == 0 {
		return
	}
	w.sb.WriteString(w.pad(depth))
	w.sb.WriteString(key)
	w.sb.WriteString(":\n")
	w.sb.WriteString(sub.sb.String())
}

// ---- scalar string quoting ---------------------------------------------

var yamlReservedWords = map[string]bool{
	"true": true, "True": true, "TRUE": true,
	"false": true, "False": true, "FALSE": true,
	"yes": true, "Yes": true, "YES": true,
	"no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
	"y": true, "Y": true, "n": true, "N": true,
	"null": true, "Null": true, "NULL": true, "~": true,
}

// quoteYAMLString returns s as a plain scalar when that's safe, otherwise
// a quoted scalar matching yaml.v3's choice of quote style: double quotes
// for values that would otherwise resolve to a different YAML type (empty,
// bool-like, number-like), single quotes for values that are unambiguously
// strings but contain characters unsafe for plain style.
func quoteYAMLString(s string) string {
	if s == "" {
		return `""`
	}
	if yamlReservedWords[s] {
		return `"` + s + `"`
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return `"` + s + `"`
	}
	if strings.ContainsAny(s, "\n\t") {
		return quoteYAMLDouble(s)
	}
	if needsSingleQuote(s) {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return s
}

func needsSingleQuote(s string) bool {
	if s[0] == ' ' || s[len(s)-1] == ' ' {
		return true
	}
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") {
		return true
	}
	if strings.Contains(s, " #") {
		return true
	}
	switch s[0] {
	case '"', '\'', '#', '&', '*', '!', '|', '>', '%', '@', '`':
		return true
	case '-', '?', ':', ',', '[', ']', '{', '}':
		return len(s) == 1 || s[1] == ' '
	}
	return false
}

func quoteYAMLDouble(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}
