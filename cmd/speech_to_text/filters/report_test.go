package filters

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/request"
)

// TestWriteReport runs the filters that need no database on a small dataset, and checks the report.
func TestWriteReport(t *testing.T) {
	verses := append(pairDataset(), textVerse(6, "ok soytan luwasa", "ok soytanluwasa"))
	settings := request.STTFilters{SpaceHyphen: true, CharPairs: true}
	res := Results{Settings: settings, PairRate: DefaultPairRate, index: make(map[int64]int)}
	for i, vs := range verses {
		res.index[vs.ScriptId] = i
	}
	res.addStage(stageBefore, verses)
	verses, res.SpaceHyphen = FilterSpaceHyphen(verses)
	res.addStage(stageSpaceHyphen, verses)
	res.Pairs = FindCharPairs(verses, res.PairRate)
	_, res.CharPairs = FilterCharPairs(verses, res.PairRate)
	res.addStage(stageCharPairs, verses)

	dir := t.TempDir()
	t.Setenv("FCBH_DATASET_TMP", dir)
	path, status := WriteReport(context.Background(), "TEST", res)
	if status != nil {
		t.Fatal(status)
	}
	if path != filepath.Join(dir, "TEST_Filters.html") {
		t.Errorf("unexpected path %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, want := range []string{
		"Filter Report: TEST",
		"1 corrected", // spaces and hyphens
		"5 corrected", // char pairs: o to u
		"U&#43;006F",  // o, in the common pairs, with + escaped by html/template
		`<span class="from">u</span> → <span class="to">o</span>`,
		`<span class="to"> </span>`, // the space of a joined word
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the report does not contain %q", want)
		}
	}
	if strings.Contains(html, `<span class="from"></span>`) {
		t.Error("an empty side of a change was shown")
	}
	if strings.Count(html, "not run") != 3 {
		t.Errorf("expected 3 filters not run, got %d", strings.Count(html, "not run"))
	}
}
