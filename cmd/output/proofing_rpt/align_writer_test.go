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

	//database, status := input.AWSS3Input(ctx, "s3://arti-output/GaryNTest/N2QAEBSP/arti/00001/database/*.db")
	database, status := input.AWSS3Input(ctx, "s3://arti-output/GaryNTest/N2MGUPNG/arti/00003/database/*.db")
	if status != nil {
		t.Fatal(status)
	}

	conn := db.NewDBAdapter(ctx, database[0].FilePath())
	output, status := Process(conn)
	fmt.Println("output", output)
}
