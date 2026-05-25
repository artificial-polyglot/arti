package fetch

import (
	"context"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"testing"
)

func TestAPIDBPClient1(t *testing.T) {
	var req request.Request
	req.BibleId = `ENGWEB`
	req.AudioData.BibleBrain.MP3_64 = true       // = dataset.MP3
	req.TextData.BibleBrain.TextPlainEdit = true // = dataset.TEXTEDIT
	req.Testament.NT = true                      // = dataset.NT
	req.Testament.OT = true
	ctx := context.Background()
	client := NewAPIDBPClient(ctx, req.BibleId)
	var info, status = client.BibleInfo()
	if status != nil {
		t.Error(`Failure in BibleInfo`, status)
	}
	client.FindFilesets(&info, req.AudioData.BibleBrain, req.TextData.BibleBrain, req.Testament)
	var identRec db.Ident
	identRec, status = client.UpdateIdent(identRec, info, req)
	if status != nil {
		t.Fatal(status)
	}
	expect := db.Ident{BibleId: `ENGWEB`, AudioOTId: ``, AudioNTId: `ENGWEBN2DA-mp3-64`,
		TextOTId: `ENGWEBO_ET`, TextNTId: `ENGWEBN_ET`, LanguageISO: `eng`, VersionCode: `WEB`, LanguageId: 6414,
		RolvId: 0, Alphabet: `Latn`, LanguageName: `English`, VersionName: `World English Bible`}
	if expect != identRec {
		t.Error("Expected:", expect, "Test:", identRec)
	}
}
