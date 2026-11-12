package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

/*
const compare_rpt_yaml = `
is_new: false
dataset_name: N2ATGMLT
username: GaryNTest
language_iso: atg
database:
    aws_s3: s3://arti-output/GaryNTest/N2ATGMLT/arti/00001/database/N2ATGMLT.db
output:
    compare_report: true
`

*/

//func TestRun(t *testing.T) {
//	log.SetOutput("stderr")
//	courier.IsCourierTest = false
//	status := run([]string{compare_rpt_yaml})
//	if status != nil {
//		t.Error(status)
//		t.Fatal(status)
//	}
//}

const username = "GaryNTest"
const inputBucket = "arti-input"
const outputBucket = "arti-output"

type testRun struct {
	name    string
	audio   string
	filters request.STTFilters
}

func TestCompareRpt(t *testing.T) {
	log.SetOutput("stderr")
	courier.IsCourierTest = false
	var filters = request.STTFilters{SpaceHyphen: true, FABoundary: true, CharPairs: true, Hallucinations: true, MissingWords: true}
	var runs []testRun
	//runs = append(runs, testRun{name: "N1SKNSEC", audio: "Kolibugan N1SKNSEC/N1SKNSEC Chapter mp3", filters: filters})
	//runs = append(runs, testRun{name: "N2ATGMLT", audio: "Arhe N2ATGMLT/N2ATGMLT Chapter mp3", filters: filters})
	//runs = append(runs, testRun{name: "N2CCPBBS", audio: "Chakma N2CCPBBS/N2CCPBBS Chapter mp3", filters: filters})
	//runs = append(runs, testRun{name: "N2MGUPNG", audio: "Magi N2MGUPNG/N2MGUPNG Chapter VOX", filters: filters})
	runs = append(runs, testRun{name: "N2QAEBSP", audio: "Dawasamu N2QAEBSP (Gospels)/N2QAEBSP Chapter VOX", filters: filters})
	//runs = append(runs, testRun{name: "N2SHNOMF", audio: "Shan N2SHNOMF/N2SHNOMF Chapter VOX", filters: filters})
	//runs = append(runs, testRun{name: "N2XNRPMS", audio: "Kangri N2XNRPMS/N2XNRPMS Chapter mp3", filters: filters})
	//runs = append(runs, testRun{name: "P2LBEBTI", audio: "Lak P2LBEBTI (Mat-Act, Rev)/P2LBEBTI Chapter mp3", filters: filters})
	//runs = append(runs, testRun{name: "N2TTSXXX", audio: "Isan N2TTSXXX", filters: filters})
	ctx := context.Background()
	s3, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		t.Fatal(status)
	}
	for _, r := range runs {
		var yaml []string
		yaml = append(yaml, "is_new: false")
		yaml = append(yaml, "dataset_name: "+r.name)
		yaml = append(yaml, "username: "+username)
		yaml = append(yaml, "language_iso: "+strings.ToLower(r.name[2:5]))
		yaml = append(yaml, "database:")
		runNum, status1 := s3.FindLastRun(outputBucket, r.name, username)
		if status1 != nil {
			t.Fatal(status)
		}
		runStr := fmt.Sprintf("%05d", runNum)
		db := "    aws_s3: s3://" + outputBucket + "/" + username + "/" + r.name + "/arti/" + runStr + "/database/" + r.name + ".db"
		yaml = append(yaml, db)
		yaml = append(yaml, "audio_data:")
		yaml = append(yaml, "    aws_s3: s3://"+inputBucket+"/"+r.audio+"/*.mp3")
		yaml = append(yaml, "stt_filters:")
		yaml = append(yaml, "    space_hyphen: "+strconv.FormatBool(filters.SpaceHyphen))
		yaml = append(yaml, "    fa_boundary: "+strconv.FormatBool(filters.FABoundary))
		yaml = append(yaml, "    char_pairs: "+strconv.FormatBool(filters.CharPairs))
		yaml = append(yaml, "    hallucinations: "+strconv.FormatBool(filters.Hallucinations))
		yaml = append(yaml, "    missing_words: "+strconv.FormatBool(filters.MissingWords))
		msg := strings.Join(yaml, "\n")
		status = run([]string{msg})
		if status != nil {
			t.Error(status)
			t.Fatal(status)
		}
	}
}
