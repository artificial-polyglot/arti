package diff

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// type Operation int
type Operation int

const (
	OpEqual Operation = iota
	OpInsert
	OpDelete
	OpReplace
)

func (op Operation) String() string {
	switch op {
	case OpEqual:
		return "equal"
	case OpInsert:
		return "insert"
	case OpDelete:
		return "delete"
	case OpReplace:
		return "replace"
	default:
		return "unknown"
	}
}

type Diff struct {
	Type    Operation
	Text    string
	Replace string
}

type CDiff struct {
	Type    Operation
	Char    rune
	Replace rune
}

func CharDiff(refText string, asrText string) []CDiff {
	var result []CDiff
	diffs := DiffReplace(refText, asrText)
	//diffs := DiffConvert(refText, asrText)
	for _, df := range diffs {
		text := []rune(df.Text)
		replace := []rune(df.Replace)
		if df.Type == OpReplace && len(replace) != len(text) {
			panic("CharDiff: replace/text rune length mismatch")
		}
		for i := range text {
			if text[i] != ' ' || df.Type == OpInsert {
				var cdiff CDiff
				if df.Type == OpReplace {
					if len(replace) != len(text) {
						panic("CharDiff: replace/text rune length mismatch")
					}
					cdiff = CDiff{Type: df.Type, Char: text[i], Replace: replace[i]}
				} else {
					cdiff = CDiff{Type: df.Type, Char: text[i]}
				}
				result = append(result, cdiff)
			}
		}
	}
	return result
}

// DiffConvert converts diffmatchpathch tree to a diff.Diff tree without modifying it.
func DiffConvert(refText string, asrText string) []Diff {
	var result []Diff
	diffs := DiffMatchPatch(refText, asrText)
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			result = append(result, Diff{Type: OpEqual, Text: diff.Text})
		case diffmatchpatch.DiffInsert:
			result = append(result, Diff{Type: OpInsert, Text: diff.Text})
		case diffmatchpatch.DiffDelete:
			result = append(result, Diff{Type: OpDelete, Text: diff.Text})
		}
	}
	return result
}

func DiffReplace(refText string, asrText string) []Diff {
	var result []Diff
	diffs := DiffMatchPatch(refText, asrText)
	var priorDelete string
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			if priorDelete != "" {
				result = append(result, Diff{Type: OpDelete, Text: priorDelete})
				priorDelete = ""
			}
			result = append(result, Diff{Type: OpEqual, Text: diff.Text})
		case diffmatchpatch.DiffInsert:
			if priorDelete != "" {
				priorDel := []rune(priorDelete)
				diffTxt := []rune(diff.Text)
				if len(priorDel) == len(diffTxt) {
					result = append(result, Diff{Type: OpReplace, Text: priorDelete, Replace: diff.Text})
				} else {
					minSize := min(len(priorDel), len(diffTxt))
					result = append(result, Diff{Type: OpReplace, Text: string(priorDel[:minSize]),
						Replace: string(diffTxt[:minSize])})
					if len(priorDel) > minSize {
						result = append(result, Diff{Type: OpDelete, Text: string(priorDel[minSize:])})
					} else {
						result = append(result, Diff{Type: OpInsert, Text: string(diffTxt[minSize:])})
					}
				}
				priorDelete = ""
			} else {
				result = append(result, Diff{Type: OpInsert, Text: diff.Text})
			}
		case diffmatchpatch.DiffDelete:
			if priorDelete != "" {
				result = append(result, Diff{Type: OpDelete, Text: priorDelete})
			}
			priorDelete = diff.Text
		}
	}
	if priorDelete != "" {
		result = append(result, Diff{Type: OpDelete, Text: priorDelete})
	}
	return result
}

func DiffMatchPatch(refText string, asrText string) []diffmatchpatch.Diff {
	diffMatch := diffmatchpatch.New()
	refText = strings.TrimSpace(refText)
	asrText = strings.TrimSpace(asrText)
	diffs := diffMatch.DiffMain(refText, asrText, false)
	diffs = diffMatch.DiffCleanupSemanticLossless(diffs)
	//diffs = diffMatch.DiffCleanupSemantic(diffs)
	//diffs = diffMatch.DiffCleanupEfficiency(diffs)
	//diffs = diffMatch.DiffCleanupMerge(diffs)
	return diffs
}
