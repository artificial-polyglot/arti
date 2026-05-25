package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/artificial-polyglot/arti/cli_misc"
	"os"
)

/*
Write method to do /download/list

Write a program that reads Sandeeps bucket in sequence,

1. Timestamps by script
2. Timestamps by verse
3. Script

If useful /timestamps find those with timestamps

Produce a file of those that have all resources, with path for each thing.
Possibly json or csv output.
*/

const (
	Bucket      = `dbp-aeneas-staging`
	LatinN1     = `Latin_N1_organized/pass_qc/`
	LatinN2     = `Latin_N2_organized/pass_qc/`
	Script      = `core_script/`
	ScriptTS    = `cue_info_text/`
	LineAeneas  = `aeneas_line_timings/`
	VerseAeneas = `aeneas_verse_timings/`
)

func main() {
	//downloadList := downloadFilestList()
	//fmt.Println(len(downloadList))
	tsMap := readTSData()
	//fmt.Println(tsMap)
	awsFind(&tsMap)
	var results []cli_misc.TSData
	for _, ts := range tsMap {
		if ts.ScriptPath != `` &&
			ts.ScriptTSPath != `` &&
			ts.LineTSPath != `` &&
			ts.VerseTSPath != `` {
			results = append(results, ts)
		}
	}
	content, err := json.MarshalIndent(results, "", "    ")
	catchErr(err)
	err = os.WriteFile(`cli_misc/find_timestamps/TestFilesetList.json`, content, 0644)
}

func readTSData() map[string]cli_misc.TSData {
	content, err := os.ReadFile(`cli_misc/find_timestamps/FilesetList.json`)
	catchErr(err)
	var results []cli_misc.TSData
	err = json.Unmarshal(content, &results)
	catchErr(err)
	//fmt.Println(results)
	var resultMap = make(map[string]cli_misc.TSData)
	for _, rec := range results {
		resultMap[rec.MediaId] = rec
	}
	return resultMap
}

func awsFind(tsMap *map[string]cli_misc.TSData) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	catchErr(err)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "us-west-2"
	})
	var prefixes = []string{LatinN1, LatinN2}
	for _, prefix := range prefixes {
		mediaList := listObjects(ctx, client, prefix)
		for _, mediaId := range mediaList {
			ts, ok := (*tsMap)[mediaId]
			if ok {
				findData(client, prefix, mediaId, &ts)
				(*tsMap)[mediaId] = ts
			}
		}
	}

}

func listObjects(ctx context.Context, client *s3.Client, prefix string) []string {
	var results []string
	list, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	catchErr(err)
	for _, item := range list.CommonPrefixes {
		directory := aws.ToString(item.Prefix)
		mediaId := directory[len(prefix) : len(directory)-1]
		results = append(results, mediaId)
	}
	return results
}

func findData(client *s3.Client, prefix string, mediaId string, ts *cli_misc.TSData) {
	newPrefix := prefix + mediaId + "/" + Script
	count := checkExists(client, newPrefix)
	if count > 0 {
		ts.ScriptPath = newPrefix
	}
	newPrefix = prefix + mediaId + "/" + ScriptTS
	count = checkExists(client, newPrefix)
	if count > 0 {
		ts.ScriptTSPath = newPrefix
	}
	newPrefix = prefix + mediaId + "/" + LineAeneas
	count = checkExists(client, newPrefix)
	if count > 0 {
		ts.LineTSPath = newPrefix
	}
	newPrefix = prefix + mediaId + "/" + VerseAeneas
	count = checkExists(client, newPrefix)
	if count > 0 {
		ts.VerseTSPath = newPrefix
		ts.Count = count
	}
}

func checkExists(client *s3.Client, prefix string) int {
	ctx := context.Background()
	list, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(Bucket),
		Prefix: aws.String(prefix),
		//Delimiter: aws.String("/"),
	})
	catchErr(err)
	//fmt.Println(list.Contents)
	return len(list.Contents)
}
