# lang_tree

AI language tools, such as speech-to-text, text-to-speech, 
and language-ident have been developed for only a small percentage of the 
known languages.  As a result when doing AI work on a language that 
none of the tools process, it is essential to find a language closely 
related, which does have AI tool support.

The lang_tree program uses a glotto log hierarch of languages to find 
related languages.  This tool is able to find related languages for eSpeak, 
mms-language-ident, mms-speech-to-text, mms-text-to-speech, 
whisper-speech-to-text, whisper-translation.

Searches are done up and down the tree to find the closest language 
that is supported by the AI tool of interest. Closeness is defined 
by a simple count of the number of nodes.  This is a naive solution. 
If someone can suggest a statistic that defines the similarity of a 
language to its children in the hierarchy, a better algorithm can be 
added.

---

Use the lang_tree executable as follows:

```
Usage: lang_tree [-v] <iso-code> <ai-tool>
optional -v is used for a detailed response
iso-code can be any iso639-3 or iso639-1 code
ai-tool can one of the following: espeak, mms_asr, mms_lid, mms_tts, whisper, omni_asr
```
---

To install this package as a go module:
> go get github.com/faithcomesbyhearing/lang_tree

To use lang_tree in a go program:
```
var tree = NewLanguageTree(ctx context.Context)
err := tree.Load()
languages, distance, err := tree.Search(iso639 string, aiTool string)
```

The AI tools supported are as follows:
* espeak
* mms_asr
* mms_lid
* mms_tts
* whisper
* omni_asr (Meta Omnilingual ASR; build/data/omni_asr.tab holds its training hours)

The search returns a slice of iso639 codes that are related and are
supported by the AI tool.  The list can be empty if none are found.
The list will contain multiple, if multiple supported languages are 
on the same level in the hierarchy.

For the user who wishes to dig deeper, there is another search method.
```
languages, distance, err := tree.DetailSearch(iso639 string, aiTool string)
```
The DetailSearch, performs the same algorithm, but returns more information.
It returns a slice of type db.Language, which is as follows:
```
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
```