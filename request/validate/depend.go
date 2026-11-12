package validate

import (
	"strings"

	"github.com/artificial-polyglot/arti/request"
)

func (r *RequestValidator) Prereq(req *request.Request) {
	if req.Timestamps.MMSAlign {
		req.Detail.Words = true
	}
}

func (r *RequestValidator) Depend(req request.Request) {
	if req.Database.AWSS3 != "" {
		if req.IsNew {
			r.errors = append(r.errors, `When database.aws_s3 is set, is_new must be false`)
		}
	}
	if !req.Timestamps.NoTimestamps {
		if req.AudioData.NoAudio {
			r.errors = append(r.errors, `Timestamps are requested, but there is no audio`)
		}
		if req.TextData.NoText && !req.Timestamps.BibleBrain {
			r.errors = append(r.errors, `Timestamps are requested, but there is no text`)
			// The need for text is not a real requirement, but the system is coded to store timestamps
			// in the scripts table, and it cannot do this unless there is text.  If this becomes
			// a problem the system could be changed to insert timestamp data without text.
		}
	}
	if req.Timestamps.MMSFAVerse || req.Timestamps.MMSAlign {
		if req.TextData.NoText {
			r.errors = append(r.errors, `Timestamp estimation requested, but there is no text data`)
		}
	}
	if !req.SpeechToText.NoSpeechToText {
		if req.AudioData.NoAudio {
			r.errors = append(r.errors, `Speech to Text is requested, but there is no audio`)
		}
		//if req.Timestamps.NoTimestamps {
		//	r.errors = append(r.errors, `Speech to Text is requested, but there are no timestamps`)
		//}
	}
	// The proofing and compare reports read the qa_align tables.  They come either from qa_align
	// in this run, or from an existing database that already has qa_align results.  Any other
	// speech_to_text runs on an *_audio copy of the database, which has no qa_align tables.
	reports := reportNames(req.Output)
	if reports != `` {
		if !req.SpeechToText.NoSpeechToText && !req.SpeechToText.QAAlign {
			r.errors = append(r.errors, reports+` requested, but speech_to_text is not qa_align`)
		} else if req.SpeechToText.NoSpeechToText && req.IsNew {
			r.errors = append(r.errors, reports+` requested, but there is no speech_to_text qa_align and no existing dataset`)
		}
	}
	if req.Compare.HTMLReport && req.SpeechToText.QAAlign {
		r.errors = append(r.errors, `compare html_report cannot be used with speech_to_text qa_align; use output compare_report`)
	}
	//	if req.AudioProof.HTMLReport {
	//		if req.IsNew {
	//			if !req.Timestamps.MMSAlign {
	//				r.errors = append(r.errors, `AudioProof is requested, but there is no mms_align`)
	//			}
	//if !req.SpeechToText.MMS {
	//	r.errors = append(r.errors, `AudioProof is requested, but there is no MMS_ASR`)
	//}
	//		} //else {
	//	if req.AudioProof.BaseDataset == "" {
	//		r.errors = append(r.errors, `AudioProof is requested on existing dataset, but there is no BaseDataset`)
	//	}
	//}
	//	}
}

// reportNames lists the requested output reports that need qa_align results.
func reportNames(o request.Output) string {
	var names []string
	if o.ProofingReport {
		names = append(names, `output proofing_report`)
	}
	if o.CompareReport {
		names = append(names, `output compare_report`)
	}
	return strings.Join(names, ` and `)
}
