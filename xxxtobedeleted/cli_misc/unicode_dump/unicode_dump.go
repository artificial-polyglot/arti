package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	filePath := filepath.Join(os.Getenv("HOME"), "Desktop", "001GEN.usx")
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// For each rune in the string
	for i, r := range string(content) {
		fmt.Printf("Position %d: '%c' (Unicode: U+%04X, Decimal: %d)\n",
			i, r, r, r)
		if i > 1000 {
			break
		}
	}
	charMap := make(map[rune]int)
	for _, r := range string(content) {
		charMap[r]++
	}
	runes := make([]rune, 0, len(charMap))
	for r := range charMap {
		runes = append(runes, r)
	}
	// Sort the runes
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})

	// Display in order
	fmt.Println("\n 001GEN chars")
	for _, r := range runes {
		if r > 127 {
			fmt.Printf("%c U+%04X : %d\n", r, r, charMap[r])
		}
	}
	fmt.Println("\nFraser Script")
	for i := 0xA4D0; i <= 0xA4FF; i++ {
		r := rune(i)
		fmt.Printf("%c U+%04X\n", r, i)
	}
}
