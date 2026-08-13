package validate

import (
	"strings"

	"github.com/artificial-polyglot/arti/request"
)

func (r *RequestValidator) Validate(req *request.Request) {
	r.checkRequired(req)
	r.checkTestament(&req.Testament)
	r.checkAudioData(&req.AudioData)
	r.checkTextData(&req.TextData)
	r.checkSpeechToText(&req.SpeechToText)
	r.checkSTTDecoder(&req.STTDecoder)
	r.checkDetail(&req.Detail)
	r.checkTimestamps(&req.Timestamps)
	r.checkTraining(&req.Training)
	r.checkCompareChoice(&req.Compare.CompareSettings.DoubleQuotes, "DoubleQuotes")
	r.checkCompareChoice(&req.Compare.CompareSettings.Apostrophe, "Apostrophe")
	r.checkCompareChoice(&req.Compare.CompareSettings.Hyphen, "Hyphen")
	r.checkDiacriticalMarks(&req.Compare.CompareSettings.DiacriticalMarks)
}

func (r *RequestValidator) checkRequired(req *request.Request) {
	if req.DatasetName == `` {
		r.errors = append(r.errors, `Required field dataset_name is empty`)
	}
	if req.BibleId == `` && req.LanguageISO == `` {
		r.errors = append(r.errors, `Required field bible_id: or language_iso: is empty`)
	}
	if req.Username == `` {
		r.errors = append(r.errors, `Required field username: is empty`)
	}
	req.DatasetName = strings.Replace(req.DatasetName, ` `, `_`, -1)
	if req.Compare.BaseDataset != `` {
		req.Compare.BaseDataset = strings.Replace(req.Compare.BaseDataset, ` `, `_`, -1)
	}
}

func (r *RequestValidator) checkTestament(req *request.Testament) {
	if !req.OT && !req.NT && len(req.NTBooks) == 0 && len(req.OTBooks) == 0 {
		req.OT = true
		req.NT = true
	}
}

// checkAudioData Is checking that no more than one item is selected.
// if none are selected, it will set the default: NoAudio
func (r *RequestValidator) checkAudioData(req *request.AudioData) {
	var count int
	if req.BibleBrain.MP3_64 {
		count += 1
	}
	if req.BibleBrain.MP3_16 {
		count += 1
	}
	if req.BibleBrain.OPUS {
		count += 1
	}
	if req.File != "" {
		count += 1
	}
	if req.AWSS3 != "" {
		count += 1
	}
	if req.POST != "" {
		count += 1
	}
	if count == 0 {
		req.NoAudio = true
	} else if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on AudioData")
	}
}

// checkTextData Is checking that no more than one item is selected.
// if none are selected, it will set the default: NoAudio
func (r *RequestValidator) checkTextData(req *request.TextData) {
	var count int
	if req.BibleBrain.TextUSXEdit {
		count += 1
	}
	if req.BibleBrain.TextPlainEdit {
		count += 1
	}
	if req.BibleBrain.TextPlain {
		count += 1
	}
	if req.File != "" {
		count += 1
	}
	if req.AWSS3 != "" {
		count += 1
	}
	if req.POST != "" {
		count += 1
	}
	if count == 0 {
		req.NoText = true
	} else if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on TextData")
	}
}

func (r *RequestValidator) checkSpeechToText(req *request.SpeechToText) {
	var count int
	if req.MMS {
		count += 1
	}
	if req.MMSAdapter {
		count += 1
	}
	if req.Wav2Vec2ASR {
		count += 1
	}
	if req.Whisper.Model.Large {
		count += 1
	}
	if req.Whisper.Model.Medium {
		count += 1
	}
	if req.Whisper.Model.Small {
		count += 1
	}
	if req.Whisper.Model.Base {
		count += 1
	}
	if req.Whisper.Model.Tiny {
		count += 1
	}
	if req.MMSASRAlign {
		count += 1
	}
	if count == 0 {
		req.NoSpeechToText = true
	} else if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on SpeechToText")
	}
}

func (r *RequestValidator) checkSTTDecoder(req *request.STTDecoder) {
	var count int
	if req.Greedy {
		count += 1
	}
	if req.Simple {
		count += 1
	}
	if req.Hotwords {
		count += 1
	}
	if req.Kenlm {
		count += 1
	}
	if count == 0 {
		req.NoSTTDecoder = true
	} else if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on STTDecoder")
	}
}

func (r *RequestValidator) checkDetail(req *request.Detail) {
	if !req.Lines && !req.Words && !req.Verses {
		req.Lines = true
	}
}

func (r *RequestValidator) checkTimestamps(req *request.Timestamps) {
	var count int
	if req.BibleBrain {
		count += 1
	}
	if req.Aeneas {
		count += 1
	}
	if req.TSBucket {
		count += 1
	}
	if req.MMSFAVerse {
		count += 1
	}
	if req.MMSAlign {
		count += 1
	}
	if count == 0 {
		req.NoTimestamps = true
	} else if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on Timestamps")
	}
}

func (r *RequestValidator) checkTraining(req *request.Training) {
	var count int
	if req.MMSAdapter.NumEpochs != 0 {
		count += 1
	}
	if req.Wav2Vec2Word.NumEpochs != 0 {
		count += 1
	}
	if count == 0 {
		req.NoTraining = true
	} else if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on Training")
	}
}

func (r *RequestValidator) checkCompareChoice(req *request.CompareChoice, fieldName string) {
	var count int
	if req.Remove {
		count += 1
	}
	if req.Normalize {
		count += 1
	}
	if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on "+fieldName)
	}
}

func (r *RequestValidator) checkDiacriticalMarks(req *request.DiacriticalChoice) {
	var count int
	if req.Remove {
		count += 1
	}
	if req.NormalizeNFC {
		count += 1
	}
	if req.NormalizeNFD {
		count += 1
	}
	if req.NormalizeNFKC {
		count += 1
	}
	if req.NormalizeNFKD {
		count += 1
	}
	if count > 1 {
		r.errors = append(r.errors, "Only 1 field can be set on DiacriticalMarks")
	}
}
