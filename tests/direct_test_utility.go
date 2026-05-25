package tests

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/controller"
	"github.com/artificial-polyglot/arti/decode_yaml"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"strings"
	"testing"
)

type CtlTest struct {
	BibleId   string
	Name      string
	TextNtId  string
	AudioNTId string
	TextType  request.MediaType
	Language  string
	Expected  int
}

func DirectTestUtility(requestYaml string, tests []CtlTest, t *testing.T) {
	ctx := context.Background()
	for _, tst := range tests {
		var req = strings.Replace(requestYaml, `{bibleId}`, tst.BibleId, 3)
		output, status := controller.CLIProcessEntry([]byte(req))
		//var control = controller.NewController(ctx, []byte(req))
		//filename, status := control.Process()
		if status != nil {
			t.Fatal(status)
		}
		if len(output.FilePaths) == 0 {
			t.Fatal("There were no output reports")
		}
		filename := output.FilePaths[0]
		fmt.Println(filename)
		numLines := NumFileLines(filename, t)
		if numLines != tst.Expected {
			t.Error(`Expected `, tst.Expected, `records, got`, numLines)
		}
		var decoder = decode_yaml.NewRequestDecoder(ctx)
		reqObj, status := decoder.Decode([]byte(req))
		if status != nil {
			t.Fatal(status)
		}
		identTest(reqObj.DatasetName, t, tst.TextType, ``,
			tst.TextNtId, ``, tst.AudioNTId, tst.Language)
	}
}
