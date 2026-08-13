package asr_align

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Gary N Griswold, Aug 12, 2026.  I think this is a failed attempt
// Delete when that is confirmed
func MergeWords(audioChars []Char, textChars []Char) error {
	referenceText := rebuildStrings(textChars)
	asrText := rebuildStrings(audioChars)

	dmp := diffmatchpatch.New()

	// Split into word slices
	refWords := strings.Fields(referenceText)
	asrWords := strings.Fields(asrText)

	// Re-join so DiffLinesToChars sees space-separated words
	// (Fields splits on any whitespace, join normalizes it)
	refJoined := strings.Join(refWords, "\n")
	asrJoined := strings.Join(asrWords, "\n")

	// Encode words as single characters for word-level diffing
	charsRef, charsAsr, wordArray := dmp.DiffLinesToChars(refJoined, asrJoined)

	// Diff the encoded strings
	diffs := dmp.DiffMain(charsRef, charsAsr, false)
	dmp.DiffCleanupSemantic(diffs)

	// Decode back to words
	diffs = dmp.DiffCharsToLines(diffs, wordArray)
	return nil
}

func rebuildStrings(chars []Char) string {
	var result []rune
	for _, c := range chars {
		result = append(result, c.Char)
	}
	return string(result)
}
