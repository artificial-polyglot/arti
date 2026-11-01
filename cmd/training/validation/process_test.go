package validation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

const username = "GaryNTest"
const inputBucket = "arti-input"
const outputBucket = "arti-output"

type run struct {
	name  string
	audio string
}

func TestProcess(t *testing.T) {
	var runs []run
	//runs = append(runs, run{name: "N1SKNSEC", audio: "Kolibugan N1SKNSEC/N1SKNSEC Chapter mp3"})
	//runs = append(runs, run{name: "N2ATGMLT", audio: "Arhe N2ATGMLT/N2ATGMLT Chapter mp3"})
	runs = append(runs, run{name: "N2CCPBBS", audio: "Chakma N2CCPBBS/N2CCPBBS Chapter mp3"})
	//runs = append(runs, run{name: "N2MGUPNG", audio: "Magi N2MGUPNG/N2MGUPNG Chapter VOX"})
	//runs = append(runs, run{name: "N2QAEBSP", audio: "Dawasamu N2QAEBSP (Gospels)/N2QAEBSP Chapter VOX"})
	//runs = append(runs, run{name: "N2SHNOMF", audio: "Shan N2SHNOMF/N2SHNOMF Chapter VOX"})
	//runs = append(runs, run{name: "N2XNRPMS", audio: "Kangri N2XNRPMS/N2XNRPMS Chapter mp3"})
	//runs = append(runs, run{name: "P2LBEBTI", audio: "Lak P2LBEBTI (Mat-Act, Rev)/P2LBEBTI Chapter mp3"})
	//runs = append(runs, run{name: "N2TTSXXX", audio: "Isan N2TTSXXX"})
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
		msg := strings.Join(yaml, "\n")
		component := courier.NewComponent(msg, "validation_unit_test")
		database, request, status2 := component.StartComponent()
		if status2 != nil {
			t.Fatal(status2)
		}
		defer database.Close()
		output, status1 := Process(database, request)
		if status1 != nil {
			t.Fatal(status1)
		}
		fmt.Println(output)
	}
}
