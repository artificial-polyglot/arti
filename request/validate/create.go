package validate

import (
	"github.com/artificial-polyglot/arti/request"
	"github.com/tidwall/gjson"
)

func CreateRequestFromHTML(json string) request.Request {
	var req request.Request
	req.IsNew = true
	req.Username = gjson.Get(json, "username").String()
	req.DatasetName = gjson.Get(json, "datasetName").String()
	req.BibleId = "" // Not used in FCBH
	req.LanguageISO = gjson.Get(json, "languageIso").String()
	req.AltLanguage = gjson.Get(json, "altLanguage").String()
	req.Priority = 3 // Default, I think
	for _, v := range gjson.Get(json, "notifyOk").Array() {
		req.NotifyOk = append(req.NotifyOk, v.String())
	}
	for _, v := range gjson.Get(json, "notifyErr").Array() {
		req.NotifyErr = append(req.NotifyErr, v.String())
	}
	req.Output.Directory = "" // Not set by HTML Form
	req.Output.CSV = false
	req.Output.JSON = false
	req.Output.Sqlite = false
	req.Testament.OT = true
	req.Testament.NT = true
	req.Database.AWSS3 = "" // Not used in HTML
	req.AudioData.AWSS3 = gjson.Get(json, "audioData").String()
	req.TextData.AWSS3 = gjson.Get(json, "textData").String()
	//dict['text_format_sfm'] = document.getElementById('text_format_sfm').checked
	//dict['text_format_usx'] = document.getElementById('text_format_usx').checked
	req.Training.RedoTraining = gjson.Get(json, "redoTraining").Bool()
	if gjson.Get(json, "training_mms_adapter").Bool() {
		req.Training.MMSAdapter.BatchMB = 4
		req.Training.MMSAdapter.NumEpochs = 16
		req.Training.MMSAdapter.LearningRate = 0.001
		req.Training.MMSAdapter.WarmupPct = 12
		req.Training.MMSAdapter.GradNormMax = 0.4
		req.SpeechToText.MMSAdapter = true
	}
	if gjson.Get(json, "training_no_training").Bool() {
		req.Training.NoTraining = true
		req.SpeechToText.MMS = true
	}
	req.Timestamps.MMSAlign = gjson.Get(json, "timestamps_mms_align").Bool()
	req.Timestamps.MMSFAVerse = gjson.Get(json, "timestamps_mms_fa_verse").Bool()
	req.AudioProof.HTMLReport = gjson.Get(json, "proofing").Bool()
	if gjson.Get(json, "compare").Bool() {
		req.Compare.HTMLReport = true
		req.Compare.GordonFilter = int(gjson.Get(json, "").Int())
		req.Compare.CompareSettings.LowerCase = true
		req.Compare.CompareSettings.RemovePromptChars = true
		req.Compare.CompareSettings.RemovePunctuation = true
		req.Compare.CompareSettings.DoubleQuotes.Remove = true
		req.Compare.CompareSettings.Apostrophe.Remove = true
		req.Compare.CompareSettings.Hyphen.Remove = true
		req.Compare.CompareSettings.DiacriticalMarks.NormalizeNFC = true
	}
	return req
}
