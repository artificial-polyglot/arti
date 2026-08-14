package validate

import (
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/request"
)

func createRequestFromMap(reqMap map[string]string) request.Request {
	var req request.Request
	req.IsNew = true
	req.Username = reqMap["username"]
	req.DatasetName = reqMap["datasetName"]
	req.BibleId = "" // Not used in FCBH
	req.LanguageISO = reqMap["languageIso"]
	req.AltLanguage = reqMap["altLanguage"]
	req.Priority = 3 // Default, I think
	req.NotifyOk = strings.Split(reqMap["notifyOk"], ",")
	req.NotifyErr = strings.Split(reqMap["notifyErr"], ",")
	req.Output.Directory = "" // Not set by HTML Form
	req.Output.CSV = false
	req.Output.JSON = false
	req.Output.Sqlite = false
	req.Testament.OT = true
	req.Testament.NT = true
	req.Database.AWSS3 = "" // Not used in HTML
	req.AudioData.AWSS3 = reqMap["audioData"]
	req.TextData.AWSS3 = reqMap["textData"]
	// text_format_sfm/text_format_usx aren't Request fields - they're read
	// directly out of reqMap by ValidateAllWASM and passed to
	// precheck.ValidateFilesWASM, which overwrites AudioData/TextData.AWSS3
	// above with a glob over the dropped files when there are any.
	req.Training.RedoTraining = reqMap["redoTraining"] == "true"
	if reqMap["training_mms_adapter"] == "true" {
		req.Training.MMSAdapter.BatchMB = 4
		req.Training.MMSAdapter.NumEpochs = 16
		req.Training.MMSAdapter.LearningRate = 0.001
		req.Training.MMSAdapter.WarmupPct = 12
		req.Training.MMSAdapter.GradNormMax = 0.4
		req.SpeechToText.MMSAdapter = true
	}
	if reqMap["training_no_training"] == "true" {
		req.Training.NoTraining = true
		req.SpeechToText.MMS = true
	}
	req.Timestamps.MMSAlign = reqMap["timestamps_mms_align"] == "true"
	req.Timestamps.MMSFAVerse = reqMap["timestamps_mms_fa_verse"] == "true"
	req.AudioProof.HTMLReport = reqMap["proofing"] == "true"
	if reqMap["compare"] == "true" {
		req.Compare.HTMLReport = true
		req.Compare.GordonFilter, _ = strconv.Atoi(reqMap["gordonFilter"])
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
