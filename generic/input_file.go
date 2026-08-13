package generic

import (
	"path/filepath"

	"github.com/artificial-polyglot/arti/request"
)

type InputFile struct {
	MediaId    string
	MediaType  request.MediaType
	Testament  string
	BookId     string // not used for text_plain
	BookSeq    string
	Chapter    int // only used for audio
	ChapterEnd int
	Verse      string // not sure how this is used but parseV4AudioFilename parses it.
	VerseEnd   string
	ScriptLine string
	Filename   string
	FileExt    string
	Directory  string
	BaseURL    string
}

func (i *InputFile) FilePath() string {
	return filepath.Join(i.Directory, i.Filename)
}
