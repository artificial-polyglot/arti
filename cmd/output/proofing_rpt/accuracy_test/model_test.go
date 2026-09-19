package accuracy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	req "github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

const USERNAME = "GaryNTest"
const INPUT_BUCKET = "arti-input"
const OUTPUT_BUCKET = "arti-output"
const TEST_DATA = "accuracy_test_%s.json"
const ACCURACY_TEST_DB = "ACCURACY_TEST.db"
const APP_NAME = "/app/runpod_arti"

type testCase struct {
	On             bool
	MediaId        string
	RunNum         string
	MediaName      string
	LanguageISO    string
	Testament      req.Testament
	AudioFilesKey  string
	SetupDBInput   string
	SetupDBLocal   string
	SetupDBOutput  string
	ResultsDBInput string
	ResultsDBLocal string
}

type wordSwitch struct {
	Word       string `json:"word"`
	FromWord   int    `json:"from_word"`
	ToWord     int    `json:"to_word"`
	FromWordId int64  `json:"from_word_id"`
	ToWordId   int64  `json:"to_word_id"`
}

func CasesForTest() []testCase {
	testament := req.Testament{NTBooks: []string{"MAT", "MRK", "LUK", "JHN"}}
	var tests []testCase
	tests = append(tests, testCase{On: true, MediaId: "N1SKNSEC", RunNum: "00004", MediaName: "Kolibugan N1SKNSEC",
		LanguageISO: "skn", Testament: testament, AudioFilesKey: "N1SKNSEC Chapter mp3/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "N2ATGMLT", RunNum: "00003", MediaName: "Arhe N2ATGMLT",
		LanguageISO: "atg", Testament: testament, AudioFilesKey: "N2ATGMLT Chapter mp3/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "N2CCPBBS", RunNum: "00005", MediaName: "Chakma N2CCPBBS",
		LanguageISO: "ccp", Testament: testament, AudioFilesKey: "N2CCPBBS Chapter mp3/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "N2MGUPNG", RunNum: "00004", MediaName: "Magi N2MGUPNG",
		LanguageISO: "mgu", Testament: testament, AudioFilesKey: "N2MGUPNG Chapter VOX/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "N2QAEBSP", RunNum: "00002", MediaName: "Dawasamu N2QAEBSP (Gospels)",
		LanguageISO: "qae", Testament: testament, AudioFilesKey: "N2QAEBSP Chapter VOX/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "N2SHNOMF", RunNum: "00003", MediaName: "Shan N2SHNOMF",
		LanguageISO: "shn", Testament: testament, AudioFilesKey: "N2SHNOMF Chapter VOX/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "N2XNRPMS", RunNum: "00010", MediaName: "Kangri N2XNRPMS",
		LanguageISO: "xnr", Testament: testament, AudioFilesKey: "N2XNRPMS Chapter mp3/*.mp3"})
	tests = append(tests, testCase{On: false, MediaId: "P2LBEBTI", RunNum: "00003", MediaName: "Lak P2LBEBTI (Mat-Act, Rev)",
		LanguageISO: "lbe", Testament: testament, AudioFilesKey: "P2LBEBTI Chapter mp3/*.mp3"})
	BuildPaths(tests)
	return tests
}

func BuildPaths(tests []testCase) {
	ctx := context.Background()
	client, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		exit(status)
	}
	for i := range tests {
		t := tests[i]
		s3Prefix := filepath.Join(USERNAME, t.MediaId, "arti")
		localPrefix := filepath.Join(os.Getenv("FCBH_DATASET_TMP"), USERNAME)
		tests[i].SetupDBInput = filepath.Join(s3Prefix, t.RunNum, "database", t.MediaId+".db")
		tests[i].SetupDBLocal = filepath.Join(localPrefix, t.MediaId+".db")
		tests[i].SetupDBOutput = filepath.Join(s3Prefix, t.RunNum, "database", ACCURACY_TEST_DB)
		nextRunNum := FindNextRunNum(client, s3Prefix)
		tests[i].ResultsDBInput = filepath.Join(s3Prefix, nextRunNum, "database", t.MediaId+".db")
		tests[i].ResultsDBLocal = filepath.Join(localPrefix, t.MediaId+"_out.db")
	}
}

func FindNextRunNum(client s3_datastore.S3Client, prefix string) string {
	keys, status := client.ListPrefixes(OUTPUT_BUCKET, prefix+"/")
	if status != nil {
		exit(status)
	}
	var maximum = 0
	for _, key := range keys {
		parts := strings.Split(key, "/")
		keyNum, err := strconv.Atoi(parts[3])
		if err != nil {
			exit(err)
		}
		if keyNum > maximum {
			maximum = keyNum
		}
	}
	maximum++ // next
	next := fmt.Sprintf("%05d", maximum)
	return next
}

func storeTestCases(tests []testCase) {
	bytes, err := json.MarshalIndent(tests, "", "  ")
	if err != nil {
		exit(err)
	}
	err = os.WriteFile("accuracy_test.json", bytes, 0644)
	if err != nil {
		exit(err)
	}
}

func retrieveTestCases() []testCase {
	var result []testCase
	bytes, err := os.ReadFile("accuracy_test.json")
	if err != nil {
		exit(err)
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		exit(err)
	}
	return result
}
