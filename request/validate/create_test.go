package validate

import (
	"context"
	"testing"

	"github.com/artificial-polyglot/arti/request"
)

func TestCreateRequestFromHTML(t *testing.T) {
	const data = `{"username":"GaryNTest","datasetName":"N2WhatEVER","textData":"s3://bucket/path/*.usx","text_format_sfm":false,"text_format_usx":true,"audioData":"s3://bucket/path/*.mp3","languageIso":"iso","altLanguage":"alt","training_mms_adapter":true,"redoTraining":true,"training_no_training":false,"timestamps_mms_align":true,"timestamps_mms_fa_verse":false,"compare":true,"proofing":true,"gordonFilter":"2","notifyOk":"ntfy/arti2","notifyErr":"ntfy/arti2"}`
	req := CreateRequestFromHTML(data)
	yaml, status := request.Encode(context.Background(), "yaml", req)
	if status != nil {
		t.Fatal(status)
	}
	println(yaml)

}
