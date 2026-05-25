package speech_to_text

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"github.com/artificial-polyglot/arti/fetch"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/read"
	"testing"
)

// EUSEABN1DA eus {eus eu Basque} XX no bible
// ENGKJVN1DA eng {eng en English}
// ENGESHN1DA eng {eng en English}
// ARZVDVN1DA arz {ara ar Arabic}
// VIEVOVN2DA vie {vie vi Vietnamese}
// ENGNIVO1DA eng {eng en English}
// ENGGIDO1DA eng {eng en English}
// ENGNKJN1DA eng {eng en English}
// PORBSPN2DA por {por pt Portuguese}
// RUSSYNN2DA rus {rus ru Russian}
// TGLPBSN1DA tgl {tgl tl Tagalog}
// ENGGIDN1DA eng {eng en English}
// RUSS76N2DA rus {rus ru Russian}
// ENGNIVN1DA eng {eng en English}
// TGLPBSN2DA tgl {tgl tl Tagalog}
// ENGNIVO2DA eng {eng en English}
// ENGESVO2DA eng {eng en English}
// ENGKJVO1DA eng {eng en English}
// ENGESVN2DA eng {eng en English}
// ENGGIDN2DA eng {eng en English}
// ENGESVN1DA eng {eng en English}
// ENGNIVN2DA eng {eng en English}
// PORARAN2DA por {por pt Portuguese}
// ENGWEBN2DA eng {eng en English}
// ENGGIDO2DA eng {eng en English}
// ENGKJVN2DA eng {eng en English}
// TGLBIBN1DA tgl {tgl tl Tagalog}
// ENGESVO1DA eng {eng en English}
// ENGKJVO2DA eng {eng en English}
// INDASVN2DA ind {ind id Indonesian}
// RUSBIBN1DA rus {rus ru Russian}
// FRABIBN1DA fra {fra fr French}
// HAKTHVN2DA hak {zho zh Chinese}
// MINLAIN2DA min {msa ms Malay (macrolanguage)}
// EUSEABN2DA eus {eus eu Basque}

func TestWhisperVs(t *testing.T) {
	type testCase struct {
		bibleId string
		mediaId string
		lang2   string
		expect  int
	}
	var tests []testCase
	tests = append(tests, testCase{bibleId: `ENGWEB`, mediaId: `ENGWEBN2DA-mp3-64`, expect: 90})
	//tests = append(tests, testCase{bibleId: `APFCMU`, mediaId: `APFCMUN2DA`, expect: 90})
	//tests = append(tests, testCase{bibleId: `DYIIBS`, mediaId: `DYIIBSN2DA`, expect: 90})
	//NO TS tests = append(tests, testCase{bibleId: "ASMDPI", mediaId: "ASMDPIN1DA-mp3-64", lang2: "as", expect: 90})
	//NO TS tests = append(tests, testCase{bibleId: "BENDPI", mediaId: "BENDPIN1DA-mp3-64", lang2: "bn", expect: 90})
	//tests = append(tests, testCase{bibleId: "ENGKJV", mediaId: "ENGKJVN1DA-opus16", lang2: "en", expect: 90})
	for _, tst := range tests {
		ctx := context.Background()
		testament := request.Testament{NTBooks: []string{`TIT`, `PHM`, `3JN`}}
		//testament := request.Testament{NTBooks: []string{`3JN`}}
		testament.BuildBookMaps()
		files, status := input.DBPDirectory(ctx, tst.bibleId, `audio`, ``, tst.mediaId, testament)
		if status != nil {
			t.Fatal(status)
		}
		var database = tst.bibleId + `_WHISPER_VS.db`
		db.DestroyDatabase(database)
		conn := db.NewDBAdapter(ctx, database)
		loadPlainText(tst.bibleId, conn, testament, t)
		loadTimestamps(tst.mediaId, conn, testament, t)
		newConn, status := conn.CopyDatabase(`_STT`)
		if status != nil {
			t.Fatal(status)
		}
		var whisp = NewWhisper(tst.bibleId, newConn, `tiny`, tst.lang2)
		status = whisp.ProcessFiles(files)
		if status != nil {
			t.Fatal(status)
		}
		count, status := newConn.CountScriptRows()
		if count != 90 {
			t.Error(`CountScriptRows count != 90`, count)
		}
		newConn.Close()
	}
}

func loadPlainText(bibleId string, conn db.DBAdapter,
	testament request.Testament, t *testing.T) {
	var status *log.Status
	req := request.Request{}
	req.BibleId = bibleId
	req.Testament = testament
	parser := read.NewDBPTextEditReader(conn, req)
	status = parser.Process()
	if status != nil {
		t.Error(status)
	}
}

func loadTimestamps(filesetId string, conn db.DBAdapter,
	testament request.Testament, t *testing.T) {
	api := fetch.NewAPIDBPTimestamps(conn, filesetId)
	ok, status := api.LoadTimestamps(testament)
	if status != nil {
		t.Error(status)
	}
	fmt.Println("Timestamps OK ", ok)
}

/*
func TestAvailTSAndWhisper(t *testing.T) {
	ctx := context.Background()
	api := fetch.NewAPIDBPTimestamps(db.DBAdapter{}, ``)
	tsMap, status := api.HavingTimestamps()
	if status.IsErr {
		t.Fatal(status)
	}
	for key, _ := range tsMap {
		lang3 := strings.ToLower(key[:3])
		sil639, status := db.FindWhisperCompatibility(ctx, lang3)
		if status.IsErr {
			t.Fatal(status)
		}
		if sil639.Lang2 != `` {
			fmt.Println(key, lang3, sil639)
		}
	}
}
*/
