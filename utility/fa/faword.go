// Package fa derives word-level forced-alignment features and failure-mode
// classifications from per-character rows, using true durations recovered from
// the inter-character Silence (which restores the timing that CTC's peaky,
// one-frame-per-character spikes otherwise hide).
//
// SilenceLong tells us how to read Silence and where words/verses/chapters end:
//
//	3 -> silence is an intra-WORD gap: part of the character's real duration
//	4 -> this char ends a WORD;    Silence is the inter-word pause
//	5 -> this char ends a VERSE;   Silence is the inter-verse pause
//	6 -> this char ends a CHAPTER; Silence runs to end of file (not a real pause)
//	(1 and 2 are unused.)
//
// The primary error detector is the MINIMUM character score (empirically the
// strongest single signal): an alignment error collapses at least one character
// toward zero, while a merely-hard-but-correct word stays uniformly mediocre.
// The duration and pause features corroborate and CLASSIFY a flag rather than
// replace it: crammed -> deletion, stretched / long pause -> extra audio.
package fa

import (
	"math"
	"sort"
)

// SilenceLong values.
const (
	SilInWord          = 3 // intra-word gap
	SilBetweenWords    = 4 // word boundary
	SilBetweenVerses   = 5 // verse boundary
	SilBetweenChapters = 6 // chapter boundary / end of file
)

func isWordEnd(sl int) bool { return sl >= SilBetweenWords } // 4, 5, or 6

// FAChar mirrors your Char2, so filling it is a field-for-field copy.
type FAChar struct {
	Char        rune
	BeginTS     float64
	EndTS       float64
	FAScore     float64
	Silence     float64
	SilenceLong int
	IsASR       bool // reserved; not used yet
}

// trueDur is a character's real footprint: its spike plus, for an in-word
// character, the intra-word gap to the next one. Boundary characters exclude
// their Silence, since that is a pause, not the character's speech.
func trueDur(c FAChar) float64 {
	d := c.EndTS - c.BeginTS
	if d < 0 {
		d = 0
	}
	if c.SilenceLong == SilInWord {
		d += c.Silence
	}
	return d
}

// WordClass is the classified state of a word.
type WordClass int

const (
	ClassOK          WordClass = iota
	BoundaryArtifact           // low score explained by contamination from an adjacent error
	SuspectDeletion            // word crammed into too little audio: likely missing from audio
	SuspectInsertion           // stretched char or over-long pause: likely extra audio
	SuspectError               // low score with no benign explanation
)

func (c WordClass) String() string {
	switch c {
	case ClassOK:
		return "OK"
	case BoundaryArtifact:
		return "BoundaryArtifact"
	case SuspectDeletion:
		return "SuspectDeletion"
	case SuspectInsertion:
		return "SuspectInsertion"
	case SuspectError:
		return "SuspectError"
	default:
		return "Unknown"
	}
}

// WordFA is the feature vector for one word.
type WordFA struct {
	NumChars int

	// scores
	MinScore        float64 // primary detector
	TrimmedMinScore float64 // min after dropping contaminated edge chars
	Mean            float64
	GeoMean         float64
	P25             float64
	TrimmedGeoMean  float64

	// durations (seconds), from true per-character duration
	ExtentSec      float64 // spoken footprint: last.EndTS - first.BeginTS
	MeanCharDurSec float64 // ExtentSec / NumChars
	MinCharDurSec  float64
	MaxCharDurSec  float64

	// boundary
	TrailingPauseSec float64 // Silence after the word
	TrailingKind     int     // SilenceLong of the word-final char (4/5/6)

	// contamination (edge ramps)
	TrimmedLead  int
	TrimmedTrail int
	LeadingRamp  bool
	TrailingRamp bool

	// baseline-relative flags (set by ClassifyWord)
	Crammed        bool
	Stretched      bool
	LongTrailPause bool

	Class WordClass
}

// FAConfig holds tunable thresholds. Duration/pause thresholds are RATIOS
// against a local (per-chapter) baseline, so they travel across languages.
type FAConfig struct {
	Epsilon float64 // log() floor

	// ramp detection
	StrongScore      float64
	RampDropFrac     float64
	MaxLeadingFrac   float64
	MinRampChars     int
	RequireShortLead bool
	LeadShortDurSec  float64

	// detection + classification
	FailScore      float64 // MinScore below this flags the word
	CrammedRatio   float64 // MeanCharDur < ratio*baseline  -> crammed (deletion)
	StretchedRatio float64 // MaxCharDur  > ratio*baseline  -> stretched (insertion)
	LongPauseRatio float64 // trailing pause > ratio*baseline pause -> insertion
}

func DefaultFAConfig() FAConfig {
	return FAConfig{
		Epsilon:          1e-6,
		StrongScore:      0.70,
		RampDropFrac:     0.60,
		MaxLeadingFrac:   0.50,
		MinRampChars:     4,
		RequireShortLead: false,
		LeadShortDurSec:  0.03,
		FailScore:        0.50,
		CrammedRatio:     0.45,
		StretchedRatio:   2.50,
		LongPauseRatio:   3.00,
	}
}

// Baseline holds local (per-chapter) norms for duration and pause.
type Baseline struct {
	MedianCharDurSec    float64
	MedianTrailPauseSec float64
}

// Word is one segmented word: its char range in the input, its span, features.
type Word struct {
	Start    int // index of first char in the input slice
	End      int // one past the last char
	BeginSec float64
	EndSec   float64
	FA       WordFA
}

// AnalyzeChars is the batteries-included entry point: pass one chapter's flat
// character stream (words delimited by SilenceLong), get back classified words.
// It segments, computes intrinsic features, derives a chapter baseline, and
// classifies each word against it.
func AnalyzeChars(chars []FAChar, cfg FAConfig) []Word {
	var words []Word
	start := 0
	for i := 0; i < len(chars); i++ {
		if isWordEnd(chars[i].SilenceLong) || i == len(chars)-1 {
			seg := chars[start : i+1]
			w := Word{Start: start, End: i + 1, FA: ComputeWordFA(seg, cfg)}
			if len(seg) > 0 {
				w.BeginSec = seg[0].BeginTS
				w.EndSec = seg[len(seg)-1].EndTS
			}
			words = append(words, w)
			start = i + 1
		}
	}
	base := ComputeBaseline(words)
	for i := range words {
		ClassifyWord(&words[i].FA, base, cfg)
	}
	return words
}

// ComputeWordFA computes the intrinsic (baseline-independent) features of one
// word from its character rows. Use it directly if you already group words.
func ComputeWordFA(chars []FAChar, cfg FAConfig) WordFA {
	var f WordFA
	n := len(chars)
	if n == 0 {
		return f
	}
	f.NumChars = n

	scores := make([]float64, n)
	tdur := make([]float64, n)
	f.MinCharDurSec = math.Inf(1)
	for i, c := range chars {
		scores[i] = c.FAScore
		d := trueDur(c)
		tdur[i] = d
		if d < f.MinCharDurSec {
			f.MinCharDurSec = d
		}
		if d > f.MaxCharDurSec {
			f.MaxCharDurSec = d
		}
	}

	f.MinScore = minOf(scores)
	f.Mean = arithMean(scores)
	f.GeoMean = geoMean(scores, cfg.Epsilon)
	sorted := append([]float64(nil), scores...)
	sort.Float64s(sorted)
	f.P25 = percentile(sorted, 0.25)

	f.ExtentSec = chars[n-1].EndTS - chars[0].BeginTS
	if f.ExtentSec < 0 {
		f.ExtentSec = 0
	}
	f.MeanCharDurSec = f.ExtentSec / float64(n)

	f.TrailingPauseSec = chars[n-1].Silence
	f.TrailingKind = chars[n-1].SilenceLong

	lead, trail := edgeContaminated(scores, tdur, cfg)
	f.TrimmedLead = lead
	f.TrimmedTrail = trail
	f.LeadingRamp = lead > 0
	f.TrailingRamp = trail > 0
	if lead+trail > 0 && lead+trail < n {
		surv := scores[lead : n-trail]
		f.TrimmedGeoMean = geoMean(surv, cfg.Epsilon)
		f.TrimmedMinScore = minOf(surv)
	} else {
		f.TrimmedGeoMean = f.GeoMean
		f.TrimmedMinScore = f.MinScore
	}
	return f
}

// ComputeBaseline derives per-chapter norms from the analyzed words.
func ComputeBaseline(words []Word) Baseline {
	var durs, pauses []float64
	for _, w := range words {
		if w.FA.MeanCharDurSec > 0 {
			durs = append(durs, w.FA.MeanCharDurSec)
		}
		if w.FA.TrailingKind == SilBetweenWords {
			pauses = append(pauses, w.FA.TrailingPauseSec)
		}
	}
	return Baseline{MedianCharDurSec: median(durs), MedianTrailPauseSec: median(pauses)}
}

// ClassifyWord sets the baseline-relative flags and the overall class.
func ClassifyWord(f *WordFA, base Baseline, cfg FAConfig) {
	if base.MedianCharDurSec > 0 {
		f.Crammed = f.MeanCharDurSec < cfg.CrammedRatio*base.MedianCharDurSec
		f.Stretched = f.MaxCharDurSec > cfg.StretchedRatio*base.MedianCharDurSec
	}
	if f.TrailingKind == SilBetweenWords && base.MedianTrailPauseSec > 0 {
		f.LongTrailPause = f.TrailingPauseSec > cfg.LongPauseRatio*base.MedianTrailPauseSec
	}

	flagged := f.MinScore < cfg.FailScore
	switch {
	case flagged && f.Crammed:
		f.Class = SuspectDeletion
	case flagged && f.Stretched:
		f.Class = SuspectInsertion
	case flagged && (f.LeadingRamp || f.TrailingRamp):
		f.Class = BoundaryArtifact
	case flagged:
		f.Class = SuspectError
	case f.LongTrailPause:
		// extra audio can leave a long pause without lowering any text score
		f.Class = SuspectInsertion
	default:
		f.Class = ClassOK
	}
}

// edgeContaminated returns leading and trailing runs of characters that look
// like spillover contamination: low score vs the healthy opposite half,
// optionally corroborated by short true duration.
func edgeContaminated(scores, tdur []float64, cfg FAConfig) (lead, trail int) {
	n := len(scores)
	if n < cfg.MinRampChars {
		return 0, 0
	}
	half := n / 2
	if strong := median(scores[half:]); strong >= cfg.StrongScore {
		lead = runFromEnd(scores, tdur, cfg, strong, true)
	}
	if strong := median(scores[:half]); strong >= cfg.StrongScore {
		trail = runFromEnd(scores, tdur, cfg, strong, false)
	}
	if lead+trail >= n {
		if lead >= trail {
			trail = 0
		} else {
			lead = 0
		}
	}
	return lead, trail
}

func runFromEnd(scores, tdur []float64, cfg FAConfig, strong float64, fromFront bool) int {
	n := len(scores)
	limit := int(float64(n) * cfg.MaxLeadingFrac)
	if limit < 1 {
		limit = 1
	}
	count := 0
	for i := 0; i < limit; i++ {
		idx := i
		if !fromFront {
			idx = n - 1 - i
		}
		low := scores[idx] < cfg.RampDropFrac*strong
		shortDur := tdur[idx] < cfg.LeadShortDurSec
		if low && (!cfg.RequireShortLead || shortDur) {
			count = i + 1
		} else {
			break
		}
	}
	return count
}

func minOf(xs []float64) float64 {
	m := math.Inf(1)
	for _, x := range xs {
		if x < m {
			m = x
		}
	}
	return m
}

func arithMean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func geoMean(xs []float64, eps float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += math.Log(math.Max(x, eps))
	}
	return math.Exp(s / float64(len(xs)))
}

func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	idx := p * float64(n-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	frac := idx - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return percentile(s, 0.5)
}
