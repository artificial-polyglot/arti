package filters

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/artificial-polyglot/arti/generic"
)

// fixture is real data from N1SKNSEC run 00003: every verse with a whole-word insert, the clip
// of each, and the stats of the whole dataset.
type fixture struct {
	Stats  Stats
	Clips  map[int64]Clip
	Verses []generic.Verse2
}

func readFixture(t *testing.T) fixture {
	var fx fixture
	b, err := os.ReadFile("testdata/n1sknsec_00003_inserts.json_test")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	return fx
}
