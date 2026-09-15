package accuracy

import req "github.com/artificial-polyglot/arti/request"

const USERNAME = "GaryNTest"
const INPUT_BUCKET = "arti-input"
const OUTPUT_BUCKET = "arti-output"
const TEST_DATA = "accuracy_test_%s.json"
const ACCURACY_TEST_DB = "ACCURACY_TEST.db"
const APP_NAME = "/app/runpod_arti"

type testCase struct {
	on            bool
	mediaId       string
	runNum        string
	mediaName     string
	languageISO   string
	testament     req.Testament
	audioFilesKey string
}

type wordSwitch struct {
	Word       string `json:"word"`
	FromWord   int    `json:"from_word"`
	ToWord     int    `json:"to_word"`
	FromWordId int64  `json:"from_word_id"`
	ToWordId   int64  `json:"to_word_id"`
}

func CasesForTest() []testCase {
	testament := req.Testament{NTBooks: []string{"PHM"}}
	var tests []testCase
	tests = append(tests, testCase{on: true, mediaId: "N1SKNSEC", runNum: "00004", mediaName: "Kolibugan N1SKNSEC",
		languageISO: "skn", testament: testament, audioFilesKey: "N1SKNSEC Chapter mp3/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "N2ATGMLT", runNum: "00003", mediaName: "Arhe N2ATGMLT",
		languageISO: "atg", testament: testament, audioFilesKey: "N2ATGMLT Chapter mp3/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "N2CCPBBS", runNum: "00005", mediaName: "Chakma N2CCPBBS",
		languageISO: "ccp", testament: testament, audioFilesKey: "N2CCPBBS Chapter mp3/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "N2MGUPNG", runNum: "00004", mediaName: "Magi N2MGUPNG",
		languageISO: "mgu", testament: testament, audioFilesKey: "N2MGUPNG Chapter VOX/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "N2QAEBSP", runNum: "00002", mediaName: "Dawasamu N2QAEBSP (Gospels)",
		languageISO: "qae", testament: testament, audioFilesKey: "N2QAEBSP Chapter VOX/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "N2SHNOMF", runNum: "00003", mediaName: "Shan N2SHNOMF",
		languageISO: "shn", testament: testament, audioFilesKey: "N2SHNOMF Chapter VOX/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "N2XNRPMS", runNum: "00010", mediaName: "Kangri N2XNRPMS",
		languageISO: "xnr", testament: testament, audioFilesKey: "N2XNRPMS Chapter mp3/*.mp3"})
	tests = append(tests, testCase{on: false, mediaId: "P2LBEBTI", runNum: "00003", mediaName: "Lak P2LBEBTI (Mat-Act, Rev)",
		languageISO: "lbe", testament: testament, audioFilesKey: "P2LBEBTI Chapter mp3/*.mp3"})
	return tests
}
