package search

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"path/filepath"
)

// https://github.com/espeak-ng/espeak-ng/blob/master/docs/languages.md?plain=1

//go:embed db/*json
var embedFile embed.FS

type LanguageTree struct {
	ctx    context.Context
	Table  []Language
	Roots  []*Language
	idMap  map[string]*Language
	isoMap map[string]*Language
}

func NewLanguageTree(ctx context.Context) LanguageTree {
	var l LanguageTree
	l.ctx = ctx
	l.idMap = make(map[string]*Language)
	l.isoMap = make(map[string]*Language)
	return l
}

func (l *LanguageTree) Load() error {
	err := l.loadTable()
	if err != nil {
		return err
	}
	err = l.buildTree()
	return err
}

func (l *LanguageTree) Search(iso639 string, toolName string) ([]string, int, error) {
	languages, distance, err := l.DetailSearch(iso639, toolName)
	if err != nil {
		return []string{}, distance, err
	}
	var results []string
	for _, lang := range languages {
		results = append(results, l.isMatch(lang, toolName))
	}
	return results, distance, nil
}

func (l *LanguageTree) DetailSearch(iso639 string, toolName string) ([]*Language, int, error) {
	var results []*Language
	var distance int
	var lang *Language
	lang, ok := l.isoMap[iso639]
	if !ok {
		err := errors.New("iso code " + iso639 + " is not known.")
		return results, distance, err
	}
	if !l.validateSearch(toolName) {
		err := errors.New("Search parameter" + toolName + "is not known")
		return results, distance, err
	}
	results, distance = l.searchRelatives(lang, toolName)
	return results, distance, nil
}

func (l *LanguageTree) loadTable() error {
	// Read json file of languages
	filename := filepath.Join("db", "language_tree.hjson")
	content, err := embedFile.ReadFile(filename)
	if err != nil {
		return err
	}
	// Parse json into Language slice
	err = json.Unmarshal(content, &l.Table)
	return err
}

func (l *LanguageTree) buildTree() error {
	// Make Map of GlottoId
	for i := range l.Table {
		lang := l.Table[i]
		l.idMap[lang.GlottoId] = &lang
	}
	// Build Tree
	var idMapCount int
	var parentIdCount int
	for glottoId, lang := range l.idMap {
		idMapCount++
		if lang.ParentId != "" {
			parentIdCount++
			parent, ok := l.idMap[lang.ParentId]
			if !ok {
				return errors.New("Missing parent id " + lang.ParentId + " is not known.")
			}
			lang.Parent = parent
			l.idMap[glottoId] = lang
			lang.Parent.Children = append(lang.Parent.Children, lang)
			l.idMap[lang.ParentId] = lang.Parent
		}
		if lang.Iso6393 != "" {
			l.isoMap[lang.Iso6393] = lang
		}
		if lang.Iso6391 != "" {
			l.isoMap[lang.Iso6391] = lang
		}
	}
	//fmt.Println("count: ", idMapCount)
	//fmt.Println("parent count: ", parentIdCount)
	// Build root
	for _, lang := range l.idMap {
		if lang.Parent == nil {
			l.Roots = append(l.Roots, lang)
		}
	}
	return nil
}

func (l *LanguageTree) searchRelatives(start *Language, search string) ([]*Language, int) {
	var finalLang, results []*Language
	var hierDown int
	var hierUp = -1
	var limit = 1000
	for limit > 0 && start != nil {
		hierUp++
		//fmt.Println("\nSearching", start.Name, search, limit)
		results, hierDown = l.descendantSearch(start, search, limit)
		//fmt.Println("hierUp", hierUp, "hierDown", hierDown, "num", len(results))
		//for _, result := range results {
		//	fmt.Println("descendentSearch lang.Name", result.Name)
		//}
		if len(results) > 0 {
			finalLang = results
			limit = hierDown - 1
		}
		start = start.Parent
	}
	return finalLang, hierUp + hierDown
}

// DescendantSearch performs a breadth-first search of the LanguageTree
func (l *LanguageTree) descendantSearch(start *Language, search string, limit int) ([]*Language, int) {
	var results []*Language
	var depth int
	if start == nil {
		return results, depth
	}
	var queue LanguageQueue
	queue.Enqueue(start, 0)
	for !queue.IsEmpty() {
		item := queue.Dequeue()
		if (len(results) > 0 && item.Depth > depth) || item.Depth > limit {
			return results, depth
		}
		depth = item.Depth
		if l.isMatch(item.Lang, search) != "" {
			results = append(results, item.Lang)
		}
		//fmt.Printf("Depth: %d, Name: %s, GlottoId: %s\n", item.Depth, item.Lang.Name, item.Lang.GlottoId)

		for _, child := range item.Lang.Children {
			queue.Enqueue(child, item.Depth+1)
		}
	}
	return results, depth
}

func (l *LanguageTree) validateSearch(search string) bool {
	switch search {
	case ESpeak:
		return true
	case MMSASR:
		return true
	case MMSLID:
		return true
	case MMSTTS:
		return true
	case Whisper:
		return true
	case OmniASR:
		return true
	default:
		return false
	}
}

func (l *LanguageTree) isMatch(lang *Language, search string) string {
	switch search {
	case ESpeak:
		return lang.ESpeak
	case MMSASR:
		return lang.MMSASR
	case MMSLID:
		return lang.MMSLID
	case MMSTTS:
		return lang.MMSTTS
	case Whisper:
		return lang.Whisper
	case OmniASR:
		return lang.OmniASR
	default:
		return ""
	}
}
