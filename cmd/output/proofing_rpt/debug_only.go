package proofing_rpt

/*
import (
	"encoding/json"
	"os"
)

type wordSwitch struct {
	Word       string `json:"word"`
	FromWord   int    `json:"from_word"`
	ToWord     int    `json:"to_word"`
	FromWordId int64  `json:"from_word_id"`
	ToWordId   int64  `json:"to_word_id"`
}

func retrieveTestCases() map[int64]wordSwitch {
	var result map[int64]wordSwitch
	bytes, err := os.ReadFile("accuracy_test/accuracy_test.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		panic(err)
	}
	return result
}

*/
