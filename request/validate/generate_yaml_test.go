package validate

import (
	"testing"

	"github.com/artificial-polyglot/arti/request"
	"gopkg.in/yaml.v3"
)

func buildTestRequest() request.Request {
	var req request.Request
	req.IsNew = true
	req.DatasetName = "N2WhatEVER"
	req.Username = "GaryNTest"
	req.BibleId = "" // always empty from the HTML form - exercises "" quoting
	req.LanguageISO = "iso"
	req.AltLanguage = "alt"
	req.NotifyOk = []string{"a@x.com", "b@y.com"}
	req.NotifyErr = []string{"c@z.com"}
	req.Testament.OT = true
	req.Testament.NT = true
	req.Testament.OTBooks = []string{"GEN", "EXO"}
	req.AudioData.AWSS3 = "s3://bucket/path/*.mp3"
	req.TextData.AWSS3 = "s3://bucket/path/*.usx"
	req.Training.RedoTraining = true
	req.Training.MMSAdapter.BatchMB = 4
	req.Training.MMSAdapter.NumEpochs = 16
	req.Training.MMSAdapter.LearningRate = 0.001
	req.Training.MMSAdapter.WarmupPct = 12
	req.Training.MMSAdapter.GradNormMax = 0.4
	req.SpeechToText.MMSAdapter = true
	req.SpeechToText.Whisper.Model.Large = true
	req.Timestamps.MMSAlign = true
	req.AudioProof.HTMLReport = true
	req.Compare.HTMLReport = true
	req.Compare.GordonFilter = 2
	req.Compare.CompareSettings.LowerCase = true
	req.Compare.CompareSettings.RemovePromptChars = true
	req.Compare.CompareSettings.RemovePunctuation = true
	req.Compare.CompareSettings.DoubleQuotes.Remove = true
	req.Compare.CompareSettings.Apostrophe.Remove = true
	req.Compare.CompareSettings.Hyphen.Remove = true
	req.Compare.CompareSettings.DiacriticalMarks.NormalizeNFC = true
	return req
}

func TestMarshalMatchesYAMLv3(t *testing.T) {
	req := buildTestRequest()

	want, err := yaml.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("Marshal output does not match yaml.Marshal.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// TestMarshalOutputSetButDirectoryEmpty exercises the case where Output is
// non-zero (so the "output:" block must render) even though Directory -
// which has no omitempty tag of its own - is empty.
func TestMarshalOutputSetButDirectoryEmpty(t *testing.T) {
	var req request.Request
	req.Output.CSV = true

	want, err := yaml.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("Marshal output does not match yaml.Marshal.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestMarshalEmptyRequestMatchesYAMLv3(t *testing.T) {
	var req request.Request

	want, err := yaml.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("Marshal output does not match yaml.Marshal.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}
