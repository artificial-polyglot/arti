package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/input/precheck"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/read/script_reader"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestWordParser(t *testing.T) {
	tests := []string{`ATIWBT_USXEDIT.db`}
	for _, database := range tests {
		ctx := context.Background()
		conn := db.NewDBAdapter(ctx, database)
		word := NewWordParser(conn)
		status := word.Parse()
		if status != nil {
			t.Fatal(status)
		}
		compareScriptAndWordsForConn(conn, database, t)
		conn.Close()
	}
}

func tokensOf(t *testing.T, text string) []wordToken {
	tokens, status := tokenizeScriptText(context.Background(), text)
	if status != nil {
		t.Fatalf("tokenizeScriptText(%q) returned unexpected error: %+v", text, status)
	}
	return tokens
}

func assertTokens(t *testing.T, text string, want [][2]string) {
	t.Helper()
	tokens := tokensOf(t, text)
	if len(tokens) != len(want) {
		t.Fatalf("tokenizeScriptText(%q) = %d tokens, want %d\ngot:  %v\nwant: %v",
			text, len(tokens), len(want), tokens, want)
	}
	for i, tok := range tokens {
		if tok.ttype != want[i][0] || string(tok.text) != want[i][1] {
			t.Errorf("token %d = {%s %q}, want {%s %q}", i, tok.ttype, string(tok.text), want[i][0], want[i][1])
		}
	}
}

func TestTokenizeCombiningMark(t *testing.T) {
	// U+09CD BENGALI SIGN VIRAMA (Mn) joins ক and ষ into a single conjunct/word.
	// unicode.IsLetter is false for it, so the old rune-by-rune parser split this
	// into three tokens: W("ক"), P(virama), W("ষ").
	assertTokens(t, "ক্ষ", [][2]string{
		{`W`, "ক্ষ"},
	})
}

func TestTokenizeMidWordPunctuation(t *testing.T) {
	assertTokens(t, `Rebeka's "well-known"`, [][2]string{
		{`W`, `Rebeka's`},
		{`S`, ` `},
		{`P`, `"`},
		{`W`, `well-known`},
		{`P`, `"`},
	})
}

func TestTokenizeTrailingPunctuationNotJoined(t *testing.T) {
	// A trailing apostrophe followed by more punctuation should not be absorbed
	// into the word - only a following letter/number joins mid-word punctuation.
	assertTokens(t, `ba, ba`, [][2]string{
		{`W`, `ba`},
		{`P`, `,`},
		{`S`, ` `},
		{`W`, `ba`},
	})
}

func TestTokenizeVerseNumber(t *testing.T) {
	tokens := tokensOf(t, `{3} In the beginning`)
	if tokens[0].ttype != `V` || string(tokens[0].text) != `{3}` || tokens[0].verseNum != 3 {
		t.Errorf("got %+v, want V token {3} with verseNum 3", tokens[0])
	}
}

func TestTokenizeVerseRangeHyphen(t *testing.T) {
	tokens := tokensOf(t, `{11}-{12} text`)
	if tokens[0].ttype != `V` || string(tokens[0].text) != `{11}-{12}` || tokens[0].verseNum != 12 {
		t.Errorf("got %+v, want V token {11}-{12} with verseNum 12", tokens[0])
	}
}

func TestTokenizeVerseRangeWithinBraces(t *testing.T) {
	tokens := tokensOf(t, `{10-11} text`)
	if tokens[0].ttype != `V` || string(tokens[0].text) != `{10-11}` || tokens[0].verseNum != 11 {
		t.Errorf("got %+v, want V token {10-11} with verseNum 11", tokens[0])
	}
}

func TestTokenizeVerseRangeUnderscore(t *testing.T) {
	tokens := tokensOf(t, `{11}_{12} text`)
	if tokens[0].ttype != `V` || string(tokens[0].text) != `{11}_{12}` || tokens[0].verseNum != 12 {
		t.Errorf("got %+v, want V token {11}_{12} with verseNum 12", tokens[0])
	}
}

func TestTokenizeMalformedVerseNumberStopsRun(t *testing.T) {
	tests := []string{
		`{abc} text`, // no digit after '{'
		`{3 text`,    // unterminated, missing '}'
		`{3}-x text`, // range separator not followed by '{'
		`{3}_`,       // truncated at end of text
	}
	for _, text := range tests {
		tokens, status := tokenizeScriptText(context.Background(), text)
		if status == nil {
			t.Errorf("tokenizeScriptText(%q) = %v, want an error that stops the run", text, tokens)
		}
	}
}

// downloadTestFiles fetches s3Path into localDir and returns it as InputFiles ready
// for precheck.ValidateFiles. s3Path is either a single object (e.g. .../foo.xlsx) or a
// "*.EXT" glob, in which case the whole directory is downloaded and then filtered down
// to entries matching the glob - SFM Text folders also contain non-SFM project files
// (e.g. Settings.xml) that setMediaType cannot classify, so the filter is required.
func downloadTestFiles(client s3_datastore.S3Client, s3Path string, localDir string) ([]generic.InputFile, *log.Status) {
	var files []generic.InputFile
	trimmed := strings.TrimPrefix(s3Path, `s3://`)
	slash := strings.Index(trimmed, `/`)
	bucket := trimmed[:slash]
	key := trimmed[slash+1:]
	if strings.Contains(key, `*`) {
		lastSlash := strings.LastIndex(key, `/`)
		prefix := key[:lastSlash+1]
		glob := strings.ToLower(key[lastSlash+1:])
		status := client.DownloadFileTree(bucket, prefix, localDir)
		if status != nil {
			return files, status
		}
		entries, err := os.ReadDir(localDir)
		if err != nil {
			return files, log.Error(context.Background(), 500, err, "Error reading downloaded files in", localDir)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			matched, err := filepath.Match(glob, strings.ToLower(entry.Name()))
			if err != nil {
				return files, log.Error(context.Background(), 500, err, "Invalid glob pattern", glob)
			}
			if matched {
				files = append(files, generic.InputFile{Directory: localDir, Filename: entry.Name()})
			}
		}
	} else {
		filename := filepath.Base(key)
		status := client.DownloadFile(bucket, key, filepath.Join(localDir, filename))
		if status != nil {
			return files, status
		}
		files = append(files, generic.InputFile{Directory: localDir, Filename: filename})
	}
	return files, nil
}

// compareScriptAndWordsForConn is compareScriptAndWords adapted to an already open
// connection, so it also works with :memory: databases that cannot be reopened by path.
func compareScriptAndWordsForConn(conn db.DBAdapter, label string, t *testing.T) {
	var count = 0
	diffMatch := diffmatchpatch.New()
	records, status := conn.SelectScripts()
	if status != nil {
		t.Fatal(label, status)
	}
	for _, rec := range records {
		sql1 := `SELECT word FROM words WHERE script_id=?`
		rows, err := conn.DB.Query(sql1, rec.ScriptId)
		if err != nil {
			t.Fatal(label, err)
		}
		var words []string
		for rows.Next() {
			var word string
			if err := rows.Scan(&word); err != nil {
				rows.Close()
				t.Fatal(label, err)
			}
			words = append(words, word)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(label, err)
		}
		wordText := strings.Join(words, ``)
		diffs := diffMatch.DiffMain(rec.ScriptText, wordText, false)
		if !isMatch(diffs) {
			ref := rec.BookId + " " + strconv.Itoa(rec.ChapterNum) + ":" + strconv.Itoa(rec.VerseNum)
			fmt.Println(label, ref, diffMatch.DiffPrettyText(diffs))
			fmt.Println("=============")
			count++
		}
	}
	if count > 0 {
		t.Errorf("%s: The script and words did not match!, num Diffs %d", label, count)
	}
}

func isMatch(diffs []diffmatchpatch.Diff) bool {
	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffDelete {
			if len(strings.TrimSpace(diff.Text)) > 0 {
				return false
			}
		}
	}
	return true
}

// TestSymmetricTest is a round-trip test: for each real S3-hosted script file (or set
// of SFM files), read it into the scripts table, run WordParser to produce the words
// table, then reconstruct each script's text by concatenating its words and verify it
// is symmetric with (identical to) the original script text.
func TestSymmetricTest1(t *testing.T) {
	var tests = []string{
		"s3://arti-input/N2CCPBBS/Text Files/SFM Text/*.SFM",
		"s3://arti-input/N2CCPBBS/Text Files/Chakma_N2CCPBBS_VR_Script.xlsx",
		"s3://arti-input/N2QAEBSP/Text Files/SFM Text/*.SFM",
		"s3://arti-input/N2QAEBSP/Text Files/Vessel Text_Dawasamu (MAT-JHN)_N2QAEBSP.xlsx",
		"s3://arti-input/N2XNRPMS/SFM Text/*.SFM",
		"s3://arti-input/N2XNRPMS/Vessel Text_Kangri_N2XNRPMS_V2.xlsx",
	}
	ctx := context.Background()
	client, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		t.Fatal(status)
	}
	testament := request.Testament{OT: true, NT: true}
	for _, tst := range tests {
		localDir := t.TempDir()
		files, status1 := downloadTestFiles(client, tst, localDir)
		if status1 != nil {
			t.Fatal(tst, status1)
		}
		files, status1 = precheck.ValidateFiles(ctx, testament, files)
		if status1 != nil {
			t.Fatal(tst, status1)
		}
		if len(files) == 0 {
			t.Fatalf("%s: no files were downloaded", tst)
		}
		conn := db.NewDBAdapter(ctx, ":memory:")
		switch files[0].MediaType {
		case request.TextScript:
			reader := script_reader.NewScriptReader(conn, testament, request.SheetColumns{})
			status1 = reader.ProcessFiles(files)
		case request.TextUSFMEdit:
			parser := NewUSFMParser(conn)
			status1 = parser.ProcessFiles(files)
		default:
			t.Fatalf("%s: unexpected media type %s", tst, files[0].MediaType)
		}
		if status1 != nil {
			t.Fatal(tst, status1)
		}
		word := NewWordParser(conn)
		status1 = word.Parse()
		if status1 != nil {
			t.Fatal(tst, status1)
		}
		compareScriptAndWordsForConn(conn, tst, t)
		conn.Close()
	}
}
