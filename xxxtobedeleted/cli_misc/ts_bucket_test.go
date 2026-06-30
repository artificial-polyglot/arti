package cli_misc

import (
	"context"
	"fmt"
	"testing"
)

// This test is not working...
func TestTSBucket(t *testing.T) {
	ctx := context.Background()
	ts := NewTSBucket(ctx)
	key := ts.GetKey(ScriptTS, `ENGWEBN2DA`, `REV`, 22)
	fmt.Println(key)
	object := ts.GetObject(TSBucketName, key)
	fmt.Println(string(object))
	timestamps := ts.GetTimestamps(ScriptTS, `ENGWEBN2DA`, `REV`, 22)
	for _, time := range timestamps {
		fmt.Println(time)
	}
}

func TestTSBucket_GetTimestamps(t *testing.T) {
	ctx := context.Background()
	ts := NewTSBucket(ctx)
	key := ts.GetKey(VerseAeneas, `ENGWEBN2DA`, `REV`, 22)
	fmt.Println(key)
	object := ts.GetObject(TSBucketName, key)
	fmt.Println(string(object))
	timestamps := ts.GetTimestamps(VerseAeneas, `ENGWEBN2DA`, `REV`, 22)
	for _, time := range timestamps {
		fmt.Println(time)
	}
}

//aws s3 ls s3://dbp-aeneas-staging/Latin_N2_organized/pass_qc/ENGWEBN2DA/cue_info_text

//aeneas_verse_timings

//aws s3 ls s3://dbp-aeneas-staging/Latin_N2_organized/pass_qc/ENGWEBN2DA/aeneas_verse_timings/
