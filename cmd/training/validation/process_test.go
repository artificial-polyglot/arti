package validation

import (
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/courier"
)

const validate_rpt_yaml = `
is_new: false 
dataset_name: N2ATGMLT
username: GaryNTest
language_iso: atg
database:
    aws_s3: s3://arti-output/GaryNTest/N2ATGMLT/arti/00001/database/N2ATGMLT.db
audio_data:
    aws_s3: s3://arti-input/Arhe N2ATGMLT/N2ATGMLT Chapter mp3/*.mp3
`

func TestProcess(t *testing.T) {
	component := courier.NewComponent(validate_rpt_yaml, "validation_unit_test")
	database, request, status := component.StartComponent()
	if status != nil {
		t.Fatal(status)
	}
	defer database.Close()
	output, status1 := Process(database, request)
	if status1 != nil {
		t.Fatal(status1)
	}
	fmt.Println(output)
	//component.FinishComponent(output, status)
}
