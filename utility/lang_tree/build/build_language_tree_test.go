package build

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/artificial-polyglot/arti/utility/lang_tree/search"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBuildLanguageTree(t *testing.T) {
	BuildLanguageTree()
}

var result []*search.Language
var count int

func TestLanguageTree_BuildTree(t *testing.T) {
	var tree = search.NewLanguageTree(context.Background())
	err := tree.Load()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("count:", len(tree.Table))
	recursiveDescent(tree.Roots)
	sort.Slice(result, func(i, j int) bool {
		return result[i].GlottoId < result[j].GlottoId
	})
	if len(tree.Table) != len(result) {
		t.Errorf("len(tree.table) = %d; actual %d", len(tree.Table), len(result))
	}
	fmt.Println("count: ", count)
	outputResult(result)
}

func recursiveDescent(langs []*search.Language) {
	for _, lang := range langs {
		result = append(result, lang)
		count++
		recursiveDescent(lang.Children)
	}
}

func outputResult(results []*search.Language) {
	bytes, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		panic(err)
	}
	filename := filepath.Join("..", "search", "db", "language_tree.hjson2")
	err = os.WriteFile(filename, bytes, 0644)
	if err != nil {
		panic(err)
	}
}
