package search

const (
	ESpeak  = "espeak"
	MMSASR  = "mms_asr"
	MMSLID  = "mms_lid"
	MMSTTS  = "mms_tts"
	Whisper = "whisper"
	OmniASR = "omni_asr"
)

type Language struct {
	GlottoId    string      `json:"id"`
	FamilyId    string      `json:"family_id"`
	ParentId    string      `json:"parent_id"`
	Name        string      `json:"name"`
	Bookkeeping bool        `json:"bookkeeping"`
	Level       string      `json:"level"` //(language, dialect, family)
	Iso6393     string      `json:"iso639_3"`
	CountryIds  string      `json:"country_ids"`
	Iso6391     string      `json:"iso639_1"`
	ESpeak      string      `json:"espeak"`
	MMSASR      string      `json:"mms_asr"`
	MMSLID      string      `json:"mms_lid"`
	MMSTTS      string      `json:"mms_tts"`
	Whisper     string      `json:"whisper"`
	OmniASR     string      `json:"omni_asr"`
	Parent      *Language   `json:"-"`
	Children    []*Language `json:"-"`
}
