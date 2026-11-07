package filters

import (
	"strings"

	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/read"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// A filter for transcripts that differ from the text only by spaces and hyphens, from compare_rpt's
// CleanupSpaceAndHyphen.  ASR often joins two words of the text, splits one, or writes a hyphen as a
// space or not at all.  Where a part of a transcript differs from the text only by spaces and
// hyphens, the transcript is given the text's version, so that it is split into the same words.
//
// Unlike CleanupSpaceAndHyphen, it changes the transcript, not the diff, and it treats a hyphen of the
// text that ASR leaves out the same as a hyphen that ASR adds.  The verses' words, chars and times are
// not changed: only ASRText.  Run it first, so that the other filters compare the same words.

// SpaceHyphenChange is one part of a transcript that was given the text's spaces and hyphens.
type SpaceHyphenChange struct {
	ScriptId int64
	LineRef  generic.VerseRef
	Text     string // the text's version, now in the transcript
	ASR      string // the transcript's version
}

func (c SpaceHyphenChange) Reason() string {
	return "[" + c.ASR + "] -> [" + c.Text + "]: spaces and hyphens"
}

// FilterSpaceHyphen returns a copy of verses with the spaces and hyphens of each ASRText changed to
// the text's, where they are the only difference, and every change made.  verses must have ASR
// transcripts, as proofing_rpt.AddASRTranscript returns them.
func FilterSpaceHyphen(verses []generic.Verse2) ([]generic.Verse2, []SpaceHyphenChange) {
	result := make([]generic.Verse2, len(verses))
	var changes []SpaceHyphenChange
	for i, vs := range verses {
		result[i] = vs
		if len(vs.Words) == 0 {
			continue
		}
		asr, verseChanges := applySpaceHyphen(vs)
		if len(verseChanges) > 0 {
			result[i].ASRText = asr
			changes = append(changes, verseChanges...)
		}
	}
	return result, changes
}

// applySpaceHyphen returns the verse's transcript with each part between two equal parts of the diff
// given the text's version, when the two differ only by spaces and hyphens.
func applySpaceHyphen(vs generic.Verse2) (string, []SpaceHyphenChange) {
	var asr strings.Builder
	var changes []SpaceHyphenChange
	var text, heard strings.Builder // the text's and the transcript's side of the current part
	endPart := func() {
		if text.Len() == 0 && heard.Len() == 0 {
			return
		}
		if stripSpaceHyphen(text.String()) == stripSpaceHyphen(heard.String()) {
			asr.WriteString(text.String())
			changes = append(changes, SpaceHyphenChange{ScriptId: vs.ScriptId, LineRef: vs.LineRef,
				Text: text.String(), ASR: heard.String()})
		} else {
			asr.WriteString(heard.String())
		}
		text.Reset()
		heard.Reset()
	}
	for _, df := range diff.DiffMatchPatch(vs.Text(), vs.ASRText) {
		switch df.Type {
		case diffmatchpatch.DiffEqual:
			endPart()
			asr.WriteString(df.Text)
		case diffmatchpatch.DiffDelete:
			text.WriteString(df.Text)
		case diffmatchpatch.DiffInsert:
			heard.WriteString(df.Text)
		}
	}
	endPart()
	return asr.String(), changes
}

func stripSpaceHyphen(s string) string {
	return strings.Map(func(r rune) rune {
		if isHyphen(r) || read.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// isHyphen is compare_rpt's IsHyphen.
func isHyphen(ch rune) bool {
	switch ch {
	case 0x002D, // hyphen-minus
		0x00AD,                                         // soft hyphen
		0x2010, 0x2011, 0x2012, 0x2013, 0x2014, 0x2015, // hyphen .. horizontal bar
		0x2212,                 // minus sign
		0xFE58, 0xFE62, 0xFE63, // small em dash, small hyphen-minus
		0xFF0D: // fullwidth hyphen-minus
		return true
	}
	return false
}
