package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/artificial-polyglot/arti/utility/lang_tree/search"
	"os"
)

func main() {
	var isoCode string
	var aiTool string
	var detail bool
	if len(os.Args) == 4 && os.Args[1] == "-v" {
		detail = true
		isoCode = os.Args[2]
		aiTool = os.Args[3]
	} else if len(os.Args) == 3 && os.Args[1] != "-v" {
		isoCode = os.Args[1]
		aiTool = os.Args[2]
	} else {
		fmt.Println("Len", len(os.Args), os.Args)
		fmt.Println()
		fmt.Println("Usage: lang_tree [-v] <iso-code> <ai-tool>")
		fmt.Println("optional -v is used for a detailed response")
		fmt.Println("iso-code can be any iso639-3 or iso639-1 code")
		fmt.Println("ai-tool can one of the following: espeak, mms_asr, mms_lid, mms_tts, whisper")
		os.Exit(1)
	}
	var tree = search.NewLanguageTree(context.Background())
	err := tree.Load()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if detail {
		langs, distance, err2 := tree.DetailSearch(isoCode, aiTool)
		if err2 != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		bytes, err3 := json.MarshalIndent(langs, "", "    ")
		if err3 != nil {
			panic(err3)
		}
		fmt.Println(string(bytes))
		fmt.Println("distance:", distance)
	} else {
		langs, distance, err2 := tree.Search(isoCode, aiTool)
		if err2 != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, lang := range langs {
			fmt.Println("language:", lang)
		}
		fmt.Println("distance:", distance)
	}
	fmt.Println()
}
