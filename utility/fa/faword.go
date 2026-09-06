// Package fa derives robust word-level forced-alignment scores and failure-mode
// classifications from per-character fa_score + begin/end timestamp rows.
//
// It is designed for a CTC forced aligner (torchaudio.functional.forced_align)
// that cannot skip a target token, so a word missing from the audio distorts the
// alignment of its neighbor. Rather than collapse each word to a single mean, we
// compute several features and classify the likely failure mode:
//
//   - Over-optimism (a mostly-wrong word rescued by a few good characters) is
//     addressed by GeoMean (log-domain mean, harsh on a single bad character)
//     and by keeping Min / P25 as separate signals.
//   - Boundary spillover (the reported problem: a present word whose LEADING
//     characters were shoved onto the missing word's frames or silence) is
//     detected as a low->high ramp and repaired by trimming the contaminated
//     leading characters *by timing/position*, not by value.
//   - A genuinely missing word shows its characters crammed into near-zero
//     duration (Compressed), a deletion signal independent of score value.
//
// Timestamps are in seconds, matching begin/end computed as frame_index *
// frame_duration, where frame_duration = (len(sample)/16000) / log_probs.shape[1].
// Rename the package/types to fit your codebase. Defaults in DefaultFAConfig are
// starting points to calibrate against your labeled data.
package fa

import (
	"math"
	"sort"
)

// CharFA is one per-character forced-alignment row for a word, in text order.
// BeginSec/EndSec are the character's alignment span in seconds.
type CharFA struct {
	Char     string  // the character (for debugging/inspection)
	Score    float64 // per-character fa_score in [0,1] = exp(char log-prob)
	BeginSec float64 // alignment begin timestamp, seconds
	EndSec   float64 // alignment end timestamp, seconds
}

// Dur is the character's duration in seconds (never negative).
func (c CharFA) Dur() float64 {
	d := c.EndSec - c.BeginSec
	if d < 0 {
		return 0
	}
	return d
}

// WordClass is the classified failure mode for a word.
type WordClass int

const (
	ClassOK          WordClass = iota // present and well-aligned
	BoundaryArtifact                  // present, but leading chars contaminated by a neighbor
	SuspectDeletion                   // characters crammed into ~no audio: likely missing from audio
	SuspectError                      // low score with no benign explanation: likely a real problem
)

func (c WordClass) String() string {
	switch c {
	case ClassOK:
		return "OK"
	case BoundaryArtifact:
		return "BoundaryArtifact"
	case SuspectDeletion:
		return "SuspectDeletion"
	case SuspectError:
		return "SuspectError"
	default:
		return "Unknown"
	}
}

// WordFA is the computed feature vector for one word.
type WordFA struct {
	Mean           float64   // arithmetic mean of char scores (your current metric)
	GeoMean        float64   // geometric mean = exp(mean(log score)); harsh on bad chars
	TrimmedGeoMean float64   // GeoMean after dropping contaminated leading chars
	Min            float64   // worst character score
	P25            float64   // 25th-percentile char score
	MinCharDur     float64   // shortest character duration (seconds)
	TrimmedLead    int       // number of leading chars judged contaminated
	LeadingRamp    bool      // spillover-victim signature present
	Compressed     bool      // deletion signature: many near-zero-duration chars
	Class          WordClass // overall classification
}

// RecommendedScore is the score to trust in QA: contamination-trimmed and
// log-domain, so it serves both the spillover and over-optimism cases.
func (f WordFA) RecommendedScore() float64 { return f.TrimmedGeoMean }

// FAConfig holds the tunable thresholds. Calibrate on labeled examples.
type FAConfig struct {
	Epsilon          float64 // floor for log() so a 0 score doesn't blow up
	StrongScore      float64 // score considered "healthy" for the back region
	RampDropFrac     float64 // leading char is contaminated if < RampDropFrac*strong
	MaxLeadingFrac   float64 // never trim more than this fraction as "leading"
	MinRampChars     int     // skip ramp detection for words shorter than this
	CrammedDurSec    float64 // char with duration <= this counts as "crammed" (~1.5 frames)
	CompressedFrac   float64 // word is Compressed if this fraction of chars are crammed
	FailScore        float64 // RecommendedScore below this is a failure
	RequireShortLead bool    // also require short duration to trim a leading char
	LeadShortDurSec  float64 // duration below this is "short" for RequireShortLead
}

func DefaultFAConfig() FAConfig {
	return FAConfig{
		Epsilon:          1e-6,
		StrongScore:      0.70,
		RampDropFrac:     0.60,
		MaxLeadingFrac:   0.50,
		MinRampChars:     4,
		CrammedDurSec:    0.03, // ~1.5 frames at a 20ms stride; set to ~1.5*frame_duration
		CompressedFrac:   0.50,
		FailScore:        0.50,
		RequireShortLead: false,
		LeadShortDurSec:  0.03,
	}
}

// ComputeWordFA turns the per-character rows of one word into its feature vector.
func ComputeWordFA(chars []CharFA, cfg FAConfig) WordFA {
	var f WordFA
	n := len(chars)
	if n == 0 {
		return f
	}

	scores := make([]float64, n)
	durs := make([]float64, n)
	minDur := math.Inf(1)
	crammed := 0
	for i, c := range chars {
		scores[i] = c.Score
		d := c.Dur()
		durs[i] = d
		if d < minDur {
			minDur = d
		}
		if d <= cfg.CrammedDurSec {
			crammed++
		}
	}

	f.Mean = arithMean(scores)
	f.GeoMean = geoMean(scores, cfg.Epsilon)

	sorted := append([]float64(nil), scores...)
	sort.Float64s(sorted)
	f.Min = sorted[0]
	f.P25 = percentile(sorted, 0.25)
	f.MinCharDur = minDur
	f.Compressed = float64(crammed)/float64(n) >= cfg.CompressedFrac

	lead := leadingContaminated(scores, durs, cfg)
	f.TrimmedLead = lead
	f.LeadingRamp = lead > 0
	if lead > 0 && lead < n {
		f.TrimmedGeoMean = geoMean(scores[lead:], cfg.Epsilon)
	} else {
		f.TrimmedGeoMean = f.GeoMean
	}

	f.Class = classify(f, cfg)
	return f
}

// leadingContaminated returns the length of the longest contiguous prefix of
// characters that look like spillover contamination: low score relative to the
// word's healthy back region (optionally corroborated by short duration).
func leadingContaminated(scores, durs []float64, cfg FAConfig) int {
	n := len(scores)
	if n < cfg.MinRampChars {
		return 0
	}
	backStart := n / 2
	strong := median(scores[backStart:])
	if strong < cfg.StrongScore {
		return 0 // back region isn't healthy; the ramp explanation doesn't apply
	}
	limit := int(float64(n) * cfg.MaxLeadingFrac)
	if limit < 1 {
		limit = 1
	}
	count := 0
	for i := 0; i < limit; i++ {
		low := scores[i] < cfg.RampDropFrac*strong
		shortDur := durs[i] < cfg.LeadShortDurSec
		if low && (!cfg.RequireShortLead || shortDur) {
			count = i + 1
		} else {
			break
		}
	}
	return count
}

func classify(f WordFA, cfg FAConfig) WordClass {
	switch {
	case f.Compressed:
		return SuspectDeletion
	case f.LeadingRamp:
		return BoundaryArtifact
	case f.RecommendedScore() < cfg.FailScore:
		return SuspectError
	default:
		return ClassOK
	}
}

func arithMean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// geoMean is exp(mean(log(x))); a single near-zero value drags it toward zero.
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

// percentile does linear interpolation on an already-sorted ascending slice.
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
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return percentile(s, 0.5)
}
