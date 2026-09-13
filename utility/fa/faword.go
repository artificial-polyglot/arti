// Package fa derives word-level forced-alignment scores from per-character
// rows, using true durations recovered from the inter-character Silence
// (which restores the timing that CTC's peaky, one-frame-per-character
// spikes otherwise hide).
//
// The primary error detector is the MINIMUM character score (empirically the
// strongest single signal): an alignment error collapses at least one
// character toward zero, while a merely-hard-but-correct word stays
// uniformly mediocre. TrimmedMinScore drops leading/trailing characters that
// look like spillover contamination from an adjacent word before taking the
// minimum.
package fa

import (
	"math"
	"sort"
)

// SilInWord marks an intra-word gap: part of the character's real duration.
const SilInWord = 3

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

// WordFA is the score vector for one word.
type WordFA struct {
	MinScore        float64 // primary detector
	TrimmedMinScore float64 // min after dropping contaminated edge chars
}

// FAConfig holds tunable thresholds for edge-contamination detection.
type FAConfig struct {
	StrongScore      float64
	RampDropFrac     float64
	MaxLeadingFrac   float64
	MinRampChars     int
	RequireShortLead bool
	LeadShortDurSec  float64
}

func DefaultFAConfig() FAConfig {
	return FAConfig{
		StrongScore:      0.70,
		RampDropFrac:     0.60,
		MaxLeadingFrac:   0.50,
		MinRampChars:     4,
		RequireShortLead: false,
		LeadShortDurSec:  0.03,
	}
}

// ComputeWordFA computes MinScore and TrimmedMinScore for one word from its
// character rows.
func ComputeWordFA(chars []FAChar, cfg FAConfig) WordFA {
	var f WordFA
	n := len(chars)
	if n == 0 {
		return f
	}

	scores := make([]float64, n)
	tdur := make([]float64, n)
	for i, c := range chars {
		scores[i] = c.FAScore
		tdur[i] = trueDur(c)
	}

	f.MinScore = minOf(scores)

	lead, trail := edgeContaminated(scores, tdur, cfg)
	if lead+trail > 0 && lead+trail < n {
		f.TrimmedMinScore = minOf(scores[lead : n-trail])
	} else {
		f.TrimmedMinScore = f.MinScore
	}
	return f
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
