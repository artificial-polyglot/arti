package validate

import "testing"

func TestJsonToMap(t *testing.T) {
	const data = `{"username":"GaryNTest","datasetName":"N2WhatEVER","textData":"s3://bucket/path/*.usx","text_format_sfm":false,"text_format_usx":true,"audioData":"s3://bucket/path/*.mp3","languageIso":"iso","altLanguage":"alt","training_mms_adapter":true,"redoTraining":true,"training_no_training":false,"timestamps_mms_align":true,"timestamps_mms_fa_verse":false,"compare":true,"proofing":true,"gordonFilter":"2","notifyOk":"ntfy/arti2","notifyErr":"ntfy/arti2"}`
	m := jsonToMap(data)
	var tests = map[string]string{
		"username":                "GaryNTest",
		"datasetName":             "N2WhatEVER",
		"textData":                "s3://bucket/path/*.usx",
		"text_format_sfm":         "false",
		"text_format_usx":         "true",
		"audioData":               "s3://bucket/path/*.mp3",
		"languageIso":             "iso",
		"altLanguage":             "alt",
		"training_mms_adapter":    "true",
		"redoTraining":            "true",
		"training_no_training":    "false",
		"timestamps_mms_align":    "true",
		"timestamps_mms_fa_verse": "false",
		"compare":                 "true",
		"proofing":                "true",
		"gordonFilter":            "2",
		"notifyOk":                "ntfy/arti2",
		"notifyErr":               "ntfy/arti2",
	}
	if len(m) != len(tests) {
		t.Errorf("expected %d keys, got %d: %v", len(tests), len(m), m)
	}
	for key, want := range tests {
		if got := m[key]; got != want {
			t.Errorf("key %q: got %q, want %q", key, got, want)
		}
	}
}

func TestJsonToMapEscapedQuote(t *testing.T) {
	const data = `{"datasetName":"He said \"hi\""}`
	m := jsonToMap(data)
	if m["datasetName"] != `He said "hi"` {
		t.Errorf(`got %q, want He said "hi"`, m["datasetName"])
	}
}
