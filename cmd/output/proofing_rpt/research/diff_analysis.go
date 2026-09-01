package research

import (
	"cmp"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/artificial-polyglot/arti/generic"
	"github.com/artificial-polyglot/arti/match/diff"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

func loadPairs(mediaId string, runNum string) string {
	objectKey := filepath.Join("GaryNTest", mediaId, "arti", runNum, "output", mediaId+"_audio_compare.json")
	localPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "GaryNTest", mediaId+"_audio_compare.json")
	client, err := s3_datastore.NewS3Client(context.Background())
	if err != nil {
		exit(err)
	}
	err = client.DownloadFile("arti-output", objectKey, localPath)
	if err != nil {
		exit(err)
	}
	return localPath
}

type sizedPair struct {
	ref      generic.VerseRef
	scriptId int
	length   int
}

func analyzeDiffRpt(dbPath string) map[generic.VerseRef]int {
	bytes, err := os.ReadFile(dbPath)
	if err != nil {
		exit(err)
	}
	var result []diff.Pair
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		exit(err)
	}
	// compute max error length on each line
	var result2 []sizedPair
	for _, p := range result {
		if p.Ref.VerseStr != "0" {
			var sized sizedPair
			sized.ref = p.Ref
			sized.scriptId = p.ScriptId()
			sized.length = p.LargestLength()
			result2 = append(result2, sized)
		}
	}
	// sort by length desc
	slices.SortFunc(result2, func(a, b sizedPair) int {
		return cmp.Or(
			cmp.Compare(b.length, a.length), // count desc
			cmp.Compare(a.scriptId, b.scriptId),
		)
	})
	// print in sorted order
	//for _, v := range result2 {
	//	println(v.ref.Description(), v.length)
	//}
	// Produce a map that contains the position for each verseRef
	var lenMap = make(map[generic.VerseRef]int)
	for i, p := range result2 {
		lenMap[p.ref] = i + 1
	}
	return lenMap
}
