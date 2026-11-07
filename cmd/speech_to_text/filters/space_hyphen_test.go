package filters

import (
	"testing"

	"github.com/artificial-polyglot/arti/generic"
)

func TestFilterSpaceHyphen(t *testing.T) {
	cases := []struct {
		name, text, asr, want string
		changes               int
	}{
		{"joined words", "ok soytan luwasa soytan", "ok soytanluwasa soytan", "ok soytan luwasa soytan", 1},
		{"split word", "pogalungan doksuꞌ", "pogalu ngan doksuꞌ", "pogalungan doksuꞌ", 1},
		{"hyphen left out", "lokow-lokow pogalungan", "lokowlokow pogalungan", "lokow-lokow pogalungan", 1},
		{"hyphen as a space", "lokow-lokow pogalungan", "lokow lokow pogalungan", "lokow-lokow pogalungan", 1},
		{"hyphen added", "sima sima ong", "sima-sima ong", "sima sima ong", 1},
		{"two places", "soytan luwasa nog lokow-lokow", "soytanluwasa nog lokow lokow", "soytan luwasa nog lokow-lokow", 2},
		{"joined, and a letter missing", "soytan luwasa", "soytanluwas", "soytan luwas", 1},
		{"space heard as a letter", "soytan luwasa", "soytanxluwasa", "soytanxluwasa", 0},
		{"missing word", "ok soytan luwasa", "ok luwasa", "ok luwasa", 0},
		{"no difference", "ok soytan", "ok soytan", "ok soytan", 0},
	}
	for _, c := range cases {
		verses := []generic.Verse2{textVerse(1, c.text, c.asr)}
		result, changes := FilterSpaceHyphen(verses)
		if result[0].ASRText != c.want {
			t.Errorf("%s: got %q, want %q", c.name, result[0].ASRText, c.want)
		}
		if len(changes) != c.changes {
			t.Errorf("%s: expected %d changes, got %v", c.name, c.changes, changes)
		}
		if verses[0].ASRText != c.asr {
			t.Errorf("%s: the input verse was changed", c.name)
		}
	}
}

func TestFilterSpaceHyphenSkipsVersesWithoutWords(t *testing.T) {
	verses := []generic.Verse2{{ScriptId: 1, ASRText: "soytanluwasa"}}
	result, changes := FilterSpaceHyphen(verses)
	if result[0].ASRText != "soytanluwasa" || len(changes) != 0 {
		t.Errorf("a verse without words was changed: %q %v", result[0].ASRText, changes)
	}
}
