package read

import (
	"strings"
	"unicode"

	"github.com/artificial-polyglot/arti/utility/safe"
)

// isSpace reports whether r separates words. It is unicode.IsSpace plus the
// zero width space U+200B, which is category Cf (not White_Space) but is used as an
// invisible word separator by Khmer, Thai, Burmese and others.
func isSpace(r rune) bool {
	return unicode.IsSpace(r) || r == '\u200B'
}

// isIgnorable reports whether r is an invisible format character that carries no
// spelling: bidi marks and embeddings, soft hyphen, word joiner, BOM/ZWNBSP.
// These are dropped before tokenizing, so they never become part of a word or
// of the training vocabulary.
func isIgnorable(r rune) bool {
	switch {
	case r == '\u00AD':
		return true // soft-hypen, indicates where words can be split, invisible
	case r == '\u2060':
		return true // word-joiner, prevents a line break, invisible
	case r == '\uFEFF':
		return true // old style word-joiner
	case r == '\u061C':
		return true // arabic right to left direction mark
	case r >= '\u200E' && r <= '\u200F': // LRM, RLM
		return true // LRM, RLM, left to right, right to left direction
	case r >= '\u202A' && r <= '\u202E': // LRE, RLE, PDF, LRO, RLO
		return true // LRE, RLE, PDF, LRO, RLO, older direction controls
	case r >= '\u2066' && r <= '\u2069': // LRI, RLI, FSI, PDI
		return true // LRI, RLI, FSI, PDI, newer isolate controls
	case r == '\u200C' || r == '\u200D': // ZERO WIDTH NON-JOINER, ZERO WIDTH JOINER
		return true // ZERO WIDTH NON-JOINER (ZWNJ), ZERO WIDTH JOINER (ZWJ)
	}
	return false
}

// stripIgnorable removes the runes that isIgnorable identifies.
func stripIgnorable(s string) string {
	return strings.Map(func(r rune) rune {
		if isIgnorable(r) {
			return -1
		}
		return r
	}, s)
}

// parseDigits converts a string of decimal digits in any script to an int. It
// returns false if s is empty or contains anything but decimal digits.
func parseDigits(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		d, ok := safe.DigitValue(r)
		if !ok {
			return 0, false
		}
		n = n*10 + d
	}
	return n, true
}
