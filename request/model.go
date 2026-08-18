package request

type Request struct {
	IsNew         bool          `yaml:"is_new" json:"is_new"`
	DatasetName   string        `yaml:"dataset_name" json:"dataset_name"`
	Username      string        `yaml:"username" json:"username"`
	BibleId       string        `yaml:"bible_id" json:"bible_id"`
	LanguageISO   string        `yaml:"language_iso" json:"language_iso"`
	AltLanguage   string        `yaml:"alt_language,omitempty" json:"alt_language,omitempty"`
	Priority      int           `yaml:"priority,omitempty" json:"priority,omitempty"`
	NotifyOk      []string      `yaml:"notify_ok" json:"notify_ok"`
	NotifyErr     []string      `yaml:"notify_err" json:"notify_err"`
	Output        Output        `yaml:"output,omitempty" json:"output,omitempty"`
	Testament     Testament     `yaml:"testament,omitempty" json:"testament,omitempty"`
	Database      Database      `yaml:"database,omitempty" json:"database,omitempty"`
	AudioData     AudioData     `yaml:"audio_data,omitempty" json:"audio_data,omitempty"`
	TextData      TextData      `yaml:"text_data,omitempty" json:"text_data,omitempty"`
	SheetColumns  SheetColumns  `yaml:"sheet_columns,omitempty" json:"sheet_columns,omitempty"`
	Timestamps    Timestamps    `yaml:"timestamps,omitempty" json:"timestamps,omitempty"`
	Training      Training      `yaml:"training,omitempty" json:"training,omitempty"`
	SpeechToText  SpeechToText  `yaml:"speech_to_text,omitempty" json:"speech_to_text,omitempty"`
	STTDecoder    STTDecoder    `yaml:"stt_decoder,omitempty" json:"stt_decoder,omitempty"`
	Detail        Detail        `yaml:"detail,omitempty" json:"detail,omitempty"`
	AudioEncoding AudioEncoding `yaml:"audio_encoding,omitempty" json:"audio_encoding,omitempty"`
	TextEncoding  TextEncoding  `yaml:"text_encoding,omitempty" json:"text_encoding,omitempty"`
	AudioProof    AudioProof    `yaml:"audio_proof,omitempty" json:"audio_proof,omitempty"`
	Compare       Compare       `yaml:"compare,omitempty" json:"compare,omitempty"`
	//	UpdateDBP     UpdateDBP     `yaml:"update_dbp,omitempty" json:"update_dbp,omitempty"`
}

// GetTestUser is used for testing when there is no full request object.
func GetTestUser() string {
	return `GaryNTest`
}

type Output struct {
	Directory string `yaml:"directory" json:"directory"`
	CSV       bool   `yaml:"csv,omitempty" json:"csv,omitempty"`
	JSON      bool   `yaml:"json,omitempty" json:"json,omitempty"`
	Sqlite    bool   `yaml:"sqlite,omitempty" json:"sqlite,omitempty"`
}

type Testament struct {
	NT      bool     `yaml:"nt,omitempty" json:"nt,omitempty"`
	NTBooks []string `yaml:"nt_books,omitempty" json:"nt_books,omitempty"`
	OT      bool     `yaml:"ot,omitempty" json:"ot,omitempty"`
	OTBooks []string `yaml:"ot_books,omitempty" json:"ot_books,omitempty"`
	otMap   map[string]bool
	ntMap   map[string]bool
}

func (t *Testament) BuildBookMaps() {
	t.otMap = make(map[string]bool)
	for _, book := range t.OTBooks {
		t.otMap[book] = true
	}
	t.ntMap = make(map[string]bool)
	for _, book := range t.NTBooks {
		t.ntMap[book] = true
	}
}

func (t *Testament) Has(ttype string, bookId string) bool {
	if ttype == `NT` {
		return t.HasNT(bookId)
	} else {
		return t.HasOT(bookId)
	}
}

func (t *Testament) HasOT(bookId string) bool {
	if t.OT {
		return true
	}
	_, ok := t.otMap[bookId]
	return ok
}

func (t *Testament) HasNT(bookId string) bool {
	if t.NT {
		return true
	}
	_, ok := t.ntMap[bookId]
	return ok
}

type Database struct {
	AWSS3 string `yaml:"aws_s3,omitempty" json:"aws_s3,omitempty"`
	File  string `yaml:"file,omitempty" json:"file,omitempty"`
}

type AudioData struct {
	BibleBrain BibleBrainAudio `yaml:"bible_brain,omitempty" json:"bible_brain,omitempty"`
	File       string          `yaml:"file,omitempty" json:"file,omitempty"`
	AWSS3      string          `yaml:"aws_s3,omitempty" json:"aws_s3,omitempty"`
	POST       string          `yaml:"post,omitempty" json:"post,omitempty"`
	NoAudio    bool            `yaml:"no_audio,omitempty" json:"no_audio,omitempty"`
}

func (a AudioData) AnyBibleBrain() bool {
	return a.BibleBrain.MP3_64 || a.BibleBrain.MP3_16 || a.BibleBrain.OPUS
}

type BibleBrainAudio struct {
	MP3_64      bool   `yaml:"mp3_64,omitempty" json:"mp3_64,omitempty"`
	MP3_16      bool   `yaml:"mp3_16,omitempty" json:"mp3_16,omitempty"`
	OPUS        bool   `yaml:"opus,omitempty" json:"opus,omitempty"`
	SetTypeCode string `yaml:"set_type_code,omitempty" json:"set_type_code,omitempty"`
}

func (b BibleBrainAudio) AudioType() (string, string) {
	var result string
	var codec string
	if b.MP3_64 {
		result = `MP3`
		codec = `64kbps`
	} else if b.MP3_16 {
		result = `MP3`
		codec = `16kbps`
	} else if b.OPUS {
		result = `OPUS`
		codec = ``
	}
	return result, codec
}

type TextData struct {
	BibleBrain BibleBrainText `yaml:"bible_brain,omitempty" json:"bible_brain,omitempty"`
	File       string         `yaml:"file,omitempty" json:"file,omitempty"`
	AWSS3      string         `yaml:"aws_s3,omitempty" json:"aws_s3,omitempty"`
	POST       string         `yaml:"post,omitempty" json:"post,omitempty"`
	NoText     bool           `yaml:"no_text,omitempty" json:"no_text,omitempty"`
}

type SheetColumns struct {
	BookCol      int `yaml:"book_col,omitempty" json:"book_col,omitempty"`
	ChapterCol   int `yaml:"chapter_col,omitempty" json:"chapter_col,omitempty"`
	VerseCol     int `yaml:"verse_col,omitempty" json:"verse_col,omitempty"`
	CharacterCol int `yaml:"character_col,omitempty" json:"character_col,omitempty"`
	ActorCol     int `yaml:"actor_col,omitempty" json:"actor_col,omitempty"`
	LineCol      int `yaml:"line_col,omitempty" json:"line_col,omitempty"`
	TextCol      int `yaml:"text_col,omitempty" json:"text_col,omitempty"`
}

func (c SheetColumns) IsSet() bool {
	if c.BookCol == 0 {
		return false
	}
	if c.ChapterCol == 0 {
		return false
	}
	if c.VerseCol == 0 {
		return false
	}
	if c.LineCol == 0 {
		return false
	}
	if c.TextCol == 0 {
		return false
	}
	return true
}

func (t TextData) AnyBibleBrain() bool {
	return t.BibleBrain.TextUSXEdit || t.BibleBrain.TextPlainEdit || t.BibleBrain.TextPlain
}

type BibleBrainText struct {
	TextUSXEdit   bool `yaml:"text_usx_edit,omitempty" json:"text_usx_edit,omitempty"`
	TextPlainEdit bool `yaml:"text_plain_edit,omitempty" json:"text_plain_edit,omitempty"`
	TextPlain     bool `yaml:"text_plain,omitempty" json:"text_plain,omitempty"`
}

type MediaType string

const (
	Audio         MediaType = "audio"
	AudioDrama    MediaType = "audio_drama"
	TextUSXEdit   MediaType = "text_usx_edit"
	TextUSFMEdit  MediaType = "text_usfm_edit"
	TextPlainEdit MediaType = "text_plain_edit"
	TextPlain     MediaType = "text_plain"
	TextScript    MediaType = "text_script"
	TextCSV       MediaType = "text_csv"
	TextSTT       MediaType = "text_stt"
	TextNone      MediaType = ""
)

func (b BibleBrainText) TextType() MediaType {
	var result MediaType
	if b.TextUSXEdit {
		result = TextUSXEdit
	} else if b.TextPlainEdit {
		result = TextPlainEdit
	} else if b.TextPlain {
		result = TextPlain
	} else {
		result = TextNone
	}
	return result
}

func (t MediaType) IsFrom(ttype string) bool {
	var result = false
	switch t {
	case TextUSXEdit:
		result = ttype == `text_usx`
	case TextPlainEdit:
		result = ttype == `text_plain`
	case TextPlain:
		result = ttype == `text_plain`
	case TextScript:
		result = ttype == `text_script`
	case TextNone:
		result = ttype == `text_none`
	}
	return result
}

type Training struct {
	RedoTraining bool       `yaml:"redo_training,omitempty" json:"redo_training,omitempty"`
	MMSAdapter   MMSAdapter `yaml:"mms_adapter,omitempty" json:"mms_adapter,omitempty"`
	Wav2Vec2Word Wav2Vec2   `yaml:"wav2vec2_word,omitempty" json:"wav2vec2_word,omitempty"`
	NoTraining   bool       `yaml:"no_training,omitempty" json:"no_training,omitempty"`
}

type MMSAdapter struct {
	BatchMB      int     `yaml:"batch_mb,omitempty" json:"batch_mb,omitempty"`
	NumEpochs    int     `yaml:"num_epochs,omitempty" json:"num_epochs,omitempty"`
	LearningRate float64 `yaml:"learning_rate,omitempty" json:"learning_rate,omitempty"`
	WarmupPct    float64 `yaml:"warmup_pct,omitempty" json:"warmup_pct,omitempty"`
	GradNormMax  float64 `yaml:"grad_norm_max,omitempty" json:"grad_norm_max,omitempty"`
}

type Wav2Vec2 struct {
	BatchMB      int     `yaml:"batch_mb,omitempty" json:"batch_mb,omitempty"`
	NumEpochs    int     `yaml:"num_epochs,omitempty" json:"num_epochs,omitempty"`
	LearningRate float64 `yaml:"learning_rate,omitempty" json:"learning_rate,omitempty"`
	WarmupPct    float64 `yaml:"warmup_pct,omitempty" json:"warmup_pct,omitempty"`
	GradNormMax  float64 `yaml:"grad_norm_max,omitempty" json:"grad_norm_max,omitempty"`
	MinAudioSec  float64 `yaml:"min_audio_sec,omitempty" json:"min_audio_sec,omitempty"`
	//LoggingSteps int     `yaml:"logging_steps,omitempty" json:"logging_steps,omitempty"`
}

type SpeechToText struct {
	MMS            bool    `yaml:"mms_asr,omitempty" json:"mms_asr,omitempty"`
	MMSAdapter     bool    `yaml:"adapter_asr,omitempty" json:"adapter_asr,omitempty"`
	Wav2Vec2ASR    bool    `yaml:"wav2vec2_asr,omitempty" json:"wav2vec2_asr,omitempty"`
	Whisper        Whisper `yaml:"whisper,omitempty" json:"whisper,omitempty"`
	MMSASRAlign    bool    `yaml:"mms_asr_align,omitempty" json:"mms_asr_align,omitempty"`
	NoSpeechToText bool    `yaml:"no_speech_to_text,omitempty" json:"no_speech_to_text,omitempty"`
}

type STTDecoder struct {
	Greedy       bool `yaml:"greedy,omitempty" json:"greedy,omitempty"`
	Simple       bool `yaml:"simple,omitempty" json:"simple,omitempty"`
	Hotwords     bool `yaml:"hotwords,omitempty" json:"hotwords,omitempty"`
	Kenlm        bool `yaml:"kenlm,omitempty" json:"kenlm,omitempty"`
	NoSTTDecoder bool `yaml:"no_stt_decoder,omitempty" json:"no_stt_decoder,omitempty"`
}

func (d STTDecoder) String() string {
	if d.Greedy {
		return "greedy"
	} else if d.Simple {
		return "simple"
	} else if d.Hotwords {
		return "hotwords"
	} else if d.Kenlm {
		return "kenlm"
	} else {
		return "greedy"
	}
}

type Whisper struct {
	Model WhisperModel `yaml:"model,omitempty" json:"model,omitempty"`
}
type WhisperModel struct {
	Large  bool `yaml:"large,omitempty" json:"large,omitempty"`
	Medium bool `yaml:"medium,omitempty" json:"medium,omitempty"`
	Small  bool `yaml:"small,omitempty" json:"small,omitempty"`
	Base   bool `yaml:"base,omitempty" json:"base,omitempty"`
	Tiny   bool `yaml:"tiny,omitempty" json:"tiny,omitempty"`
}

func (w WhisperModel) String() string {
	var result string
	if w.Large {
		result = `large`
	} else if w.Medium {
		result = `medium`
	} else if w.Small {
		result = `small`
	} else if w.Base {
		result = `base`
	} else if w.Tiny {
		result = `tiny`
	}
	return result
}

type Detail struct {
	Lines  bool `yaml:"lines,omitempty" json:"lines,omitempty"`
	Verses bool `yaml:"verses,omitempty" json:"verses,omitempty"`
	Words  bool `yaml:"words,omitempty" json:"words,omitempty"`
}

type Timestamps struct {
	BibleBrain   bool `yaml:"bible_brain,omitempty" json:"bible_brain,omitempty"`
	Aeneas       bool `yaml:"aeneas,omitempty" json:"aeneas,omitempty"`
	TSBucket     bool `yaml:"ts_bucket,omitempty" json:"ts_bucket,omitempty"`
	MMSFAVerse   bool `yaml:"mms_fa_verse,omitempty" json:"mms_fa_verse,omitempty"`
	MMSAlign     bool `yaml:"mms_align,omitempty" json:"mms_align,omitempty"`
	NoTimestamps bool `yaml:"no_timestamps,omitempty" json:"no_timestamps,omitempty"`
}

type AudioEncoding struct {
	MFCC       bool `yaml:"mfcc,omitempty" json:"mfcc,omitempty"`
	NoEncoding bool `yaml:"no_encoding,omitempty" json:"no_encoding,omitempty"`
}

type TextEncoding struct {
	FastText   bool `yaml:"fast_text,omitempty" json:"fast_text,omitempty"`
	NoEncoding bool `yaml:"no_encoding,omitempty" json:"no_encoding,omitempty"`
}

type AudioProof struct {
	HTMLReport bool `yaml:"html_report,omitempty" json:"html_report,omitempty"`
}

type Compare struct {
	HTMLReport      bool            `yaml:"html_report,omitempty" json:"html_report,omitempty"`
	BaseDataset     string          `yaml:"base_dataset,omitempty" json:"base_dataset,omitempty"`
	GordonFilter    int             `yaml:"gordon_filter,omitempty" json:"gordon_filter,omitempty"`
	CompareSettings CompareSettings `yaml:"compare_settings,omitempty" json:"compare_settings,omitempty"`
}

type CompareSettings struct {
	LowerCase         bool              `yaml:"lower_case,omitempty" json:"lower_case,omitempty"`
	RemovePromptChars bool              `yaml:"remove_prompt_chars,omitempty" json:"remove_prompt_chars,omitempty"`
	RemovePunctuation bool              `yaml:"remove_punctuation,omitempty" json:"remove_punctuation,omitempty"`
	DoubleQuotes      CompareChoice     `yaml:"double_quotes,omitempty" json:"double_quotes,omitempty"`
	Apostrophe        CompareChoice     `yaml:"apostrophe,omitempty" json:"apostrophe,omitempty"`
	Hyphen            CompareChoice     `yaml:"hyphen,omitempty" json:"hyphen,omitempty"`
	DiacriticalMarks  DiacriticalChoice `yaml:"diacritical_marks,omitempty" json:"diacritical_marks,omitempty"`
}

type CompareChoice struct {
	Remove    bool `yaml:"remove,omitempty" json:"remove,omitempty"`
	Normalize bool `yaml:"normalize,omitempty" json:"normalize,omitempty"`
}

type DiacriticalChoice struct {
	Remove        bool `yaml:"remove,omitempty" json:"remove,omitempty"`
	NormalizeNFC  bool `yaml:"normalize_nfc,omitempty" json:"normalize_nfc,omitempty"`
	NormalizeNFD  bool `yaml:"normalize_nfd,omitempty" json:"normalize_nfd,omitempty"`
	NormalizeNFKC bool `yaml:"normalize_nfkc,omitempty" json:"normalize_nfkc,omitempty"`
	NormalizeNFKD bool `yaml:"normalize_nfkd,omitempty" json:"normalize_nfkd,omitempty"`
}

//type UpdateDBP struct {
//	Timestamps         string `yaml:"timestamps,omitempty" json:"timestamps,omitempty"`
//	HLS                string `yaml:"hls,omitempty" json:"hls,omitempty"`
//	CopyTimestampsFrom string `yaml:"copy_timestamps_from,omitempty" json:"copy_timestamps_from,omitempty"`
//}
