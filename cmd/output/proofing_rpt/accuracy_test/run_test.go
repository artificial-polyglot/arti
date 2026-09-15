////go:build runpod
// The //go is preventing compilation, because this test submits jobs to runpod, which costs money
// To turn it on change the // to ///

package accuracy

import (
	"fmt"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/dispatch/runpod"
	"github.com/artificial-polyglot/arti/request"
	"gopkg.in/yaml.v3"
)

/*
This test generates a yaml file for a run, and generates a shell file that will submit it to
*/

func TestRun(t *testing.T) {
	tests := CasesForTest()
	for _, tst := range tests {
		if tst.on {
			yamlString := CreateYaml(tst)
			SubmitToRunpod(APP_NAME, tst.mediaId, yamlString)
		}
	}
}

func CreateYaml(test testCase) string {
	var req request.Request
	req.IsNew = false
	req.DatasetName = test.mediaId
	req.Username = USERNAME
	req.LanguageISO = test.languageISO
	req.Priority = 3
	req.NotifyOk = []string{"ntfy/arti2"}
	req.NotifyErr = []string{"ntfy/arti2"}
	req.Testament = test.testament
	database := fmt.Sprintf("s3://%s/%s/%s/arti/%s/database/%s", OUTPUT_BUCKET, USERNAME, test.mediaId,
		test.runNum, ACCURACY_TEST_DB)
	req.Database.AWSS3 = database
	req.AudioData.AWSS3 = fmt.Sprintf("s3://%s/%s/%s", INPUT_BUCKET, test.mediaName, test.audioFilesKey)
	//req.Detail.Lines = true
	//req.Detail.Words = true
	req.AudioProof.HTMLReport = true
	//req.Compare.HTMLReport = true
	//req.Compare.CompareSettings.LowerCase = true
	//req.Compare.CompareSettings.RemovePromptChars = true
	//req.Compare.CompareSettings.RemovePunctuation = true
	//req.Compare.CompareSettings.DoubleQuotes.Remove = true
	//req.Compare.CompareSettings.Apostrophe.Remove = true
	//req.Compare.CompareSettings.Hyphen.Remove = true
	//req.Compare.CompareSettings.DiacriticalMarks.NormalizeNFC = true
	bytes, err := yaml.Marshal(req)
	if err != nil {
		exit(err)
	}
	return string(bytes)
}

func SubmitToRunpod(appName string, mediaId string, yamlString string) {
	client, err := runpod.New()
	if err != nil {
		exit(err)
	}

	jobID, err := client.Submit(appName, yamlString)
	if err != nil {
		exit(err)
	}
	fmt.Printf("submitted job %s\n", jobID)

	err = runpod.Notify(fmt.Sprintf("Task %s submitted job: %s", mediaId, jobID))
	if err != nil {
		exit(err)
	}
}
