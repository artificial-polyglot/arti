package mms_align

import (
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
)

func TestPadVerses(t *testing.T) {
	var m MMSAlign
	chars := []generic.Char{{}}
	chapter := [][]db.Audio{
		{{ScriptId: 1, BeginTS: 0.5, EndTS: 1, Chars: chars}, {ScriptId: 1, BeginTS: 1.2, EndTS: 2, Chars: chars}},
		// a verse starting with a word that has no FA chars, timed at the previous word's end
		{{ScriptId: 2, BeginTS: 2, EndTS: 2}, {ScriptId: 2, BeginTS: 3, EndTS: 4, Chars: chars}},
		{{ScriptId: 3, BeginTS: 4.1, EndTS: 5, Chars: chars}},
	}
	got := m.padVerses(m.summarizeByVerse(chapter))
	if got[0].BeginTS != 0.3 || got[0].EndTS != 2.2 {
		t.Errorf("full padding on both sides: %+v", got[0])
	}
	if got[1].BeginTS != 2.8 || got[1].EndTS != 4.05 {
		t.Errorf("speech must start at the first aligned word, and end at half the gap: %+v", got[1])
	}
	if got[2].BeginTS != 4.05 || got[2].EndTS != 5.2 {
		t.Errorf("start at half the gap, full padding at the end of the file: %+v", got[2])
	}
	if got[0].PlayBeginTS != 0 || got[0].PlayEndTS != 2.5 || got[1].PlayBeginTS != 2.5 ||
		got[1].PlayEndTS != 4.05 || got[2].PlayBeginTS != 4.05 {
		t.Errorf("play times must be midpoints, starting at 0: %+v", got)
	}
}
