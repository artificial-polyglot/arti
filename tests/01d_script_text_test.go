package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/courier"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/request"
)

const scriptTextScript = `is_new: yes
dataset_name: 01d_script_text_{bibleId}
bible_id: {bibleId}
username: Tests
output:
  directory: ~/Downloads
  csv: yes
text_data:
  file: ../read/script_reader/test_data/CORE_Scr_1065p_1Eng__14_Spkr_Tajik_N2_TGK_IBT_arti.xlsx
`

func TestScriptTextDirect(t *testing.T) {
	courier.IsCourierTest = true
	var tests []SqliteTest
	tests = append(tests, SqliteTest{"SELECT count(*) FROM scripts", 7668})
	testName := strings.Replace(scriptTextScript, "{bibleId}", "TGKIBT", -1)
	DirectSqlTest(testName, tests, t)
}

func TestScriptTextScriptAPI(t *testing.T) {
	var cases []APITest
	cases = append(cases, APITest{BibleId: `ATIWBT`, Expected: 9747})
	APITestUtility(scriptTextScript, cases, t)
}

func TestScriptTextScript(t *testing.T) {
	var bibleId = `ATIWBT`
	ctx := context.Background()
	var req = strings.Replace(scriptTextScript, `{bibleId}`, bibleId, 2)
	var control = controller.NewController(ctx, []byte(req))
	filename, status := control.Process()
	if status != nil {
		t.Fatal(status)
	}
	fmt.Println("Filename:", filename)
	conn := db.NewDBAdapter(context.TODO(), filename)
	count, status := conn.CountScriptRows()
	if status != nil {
		t.Fatal(status)
	}
	var expected = 9747
	if count != expected {
		t.Error(`Expected `, expected, `records, got`, count)
	}
	identTest(`ScriptTextScript_`+bibleId, t, request.TextScript, ``,
		`ATIWBTN2ST`, ``, ``, `ati`)
}
