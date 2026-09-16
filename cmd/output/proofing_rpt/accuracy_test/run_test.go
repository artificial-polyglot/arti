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
	tests := retrieveTestCases()
	for _, tst := range tests {
		if tst.On {
			yamlString := CreateYaml(tst)
			SubmitToRunpod(APP_NAME, tst.MediaId, yamlString)
		}
	}
}

func CreateYaml(test testCase) string {
	var req request.Request
	req.IsNew = false
	req.DatasetName = test.MediaId
	req.Username = USERNAME
	req.LanguageISO = test.LanguageISO
	req.Priority = 3
	req.NotifyOk = []string{"ntfy/arti2"}
	req.NotifyErr = []string{"ntfy/arti2"}
	req.Testament = test.Testament
	req.Database.AWSS3 = "s3://" + OUTPUT_BUCKET + "/" + test.SetupDBOutput
	req.AudioData.AWSS3 = fmt.Sprintf("s3://%s/%s/%s", INPUT_BUCKET, test.MediaName, test.AudioFilesKey)
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
