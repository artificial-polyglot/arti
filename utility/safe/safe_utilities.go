package safe

import (
	"strconv"
	"unicode"
)

// SafeVerseNum returns a numeric value for a string by ignoring alpha characters without error
func SafeVerseNum(number string) int {
	var result []rune
	for _, chr := range number {
		if chr >= '0' && chr <= '9' {
			result = append(result, chr)
		} else {
			break
		}
	}
	num, _ := strconv.Atoi(string(result))
	return num
}

// SafeStringJoin preserve existing whitespace, while ensuring that strings are joined with whitespace between
func SafeStringJoin(texts []string) string {
	if len(texts) == 0 {
		return ""
	}
	if len(texts) == 1 {
		return texts[0]
	}
	var openPunctMap = map[rune]bool{'(': true, '[': true,
		'\u2018': true, // opening single quote mark
		'\u201C': true, // opening double quote
		'\u2039': true, // something like <
		'\u00AB': true, // something like <<
	}
	var endPunctMap = map[rune]bool{'?': true, '.': true, ',': true, ':': true, ';': true, ')': true, ']': true,
		'\u2019': true, // closing single quote mark
		'\u201D': true, // closing double quote
		'\u201E': true, // closing low double quote
		'\u203A': true, // something like >
		'\u00BB': true, // something like >>
	}
	var result []rune
	var lastIsAlpha = false
	for _, txt := range texts {
		sc := []rune(txt)
		_, isEndPunct := endPunctMap[sc[0]]
		beginSpace := unicode.IsSpace(sc[0]) || isEndPunct
		if lastIsAlpha && !beginSpace {
			result = append(result, ' ')
		}
		result = append(result, sc...)
		lastChar := sc[len(sc)-1]
		_, isOpenPunct := openPunctMap[lastChar]
		lastIsAlpha = !(unicode.IsSpace(lastChar) || isOpenPunct)
	}
	return string(result)
}
