package proofing_rpt

import (
	"context"
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
)

func TestAlignWriter(t *testing.T) {
	ctx := context.Background()
	var tests []string
	tests = append(tests, "/GaryNTest/N1SKNSEC/arti/00004/database/*.db")
	//tests = append(tests, "/GaryNTest/N2ATGMLT/arti/00003/database/*.db")
	//tests = append(tests, "/GaryNTest/N2CCPBBS/arti/00005/database/*.db")
	//tests = append(tests, "/GaryNTest/N2MGUPNG/arti/00004/database/*.db")
	//tests = append(tests, "/GaryNTest/N2QAEBSP/arti/00002/database/*.db")
	//tests = append(tests, "/GaryNTest/N2SHNOMF/arti/00002/database/*.db")
	//tests = append(tests, "/GaryNTest/N2XNRPMS/arti/00010/database/*.db")
	//tests = append(tests, "/GaryNTest/P2LBEBTI/arti/00002/database/*.db")
	for _, tt := range tests {
		//	dbURL := "s3://arti-output" + t
		//database, status := input.AWSS3Input(ctx, "s3://arti-output/GaryNTest/N2QAEBSP/arti/00001/database/*.db")
		database, status := input.AWSS3Input(ctx, "s3://arti-output"+tt)
		if status != nil {
			t.Fatal(status)
		}
		conn := db.NewDBAdapter(ctx, database[0].FilePath())
		output, status := Process(conn)
		fmt.Println("output", output)
	}
}
