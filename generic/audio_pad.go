package generic

// AudioPadSec is the context, in seconds, added on each side of the speech of every training
// and ASR sample, whether a verse or a phrase.  It is limited to half the gap to the
// neighbouring sample in the same audio file, so that neighbouring samples never overlap.
// Verses and phrases use the same padding so that a model trained on phrases hears the same
// lead-in and tail-off as the verses that ASR is run on.
const AudioPadSec = 0.20

// PadBegin returns the padded start of a sample whose speech starts at beginTS, when the
// previous sample's speech ends at prevEndTS.  hasPrev is false at the start of an audio file.
func PadBegin(beginTS float64, prevEndTS float64, hasPrev bool) float64 {
	result := beginTS - AudioPadSec
	if hasPrev {
		result = max(result, (prevEndTS+beginTS)/2)
	}
	return max(result, 0.0)
}

// PadEnd returns the padded end of a sample whose speech ends at endTS, when the next sample's
// speech starts at nextBeginTS.  hasNext is false at the end of an audio file.
func PadEnd(endTS float64, nextBeginTS float64, hasNext bool) float64 {
	result := endTS + AudioPadSec
	if hasNext {
		result = min(result, (endTS+nextBeginTS)/2)
	}
	return result
}
