package mms

import (
	"context"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/lang_tree/search"
	"os"
	"path/filepath"
	"strings"
)

// HasLocalAdapter reports whether a trained MMS adapter for lang already
// exists on local disk under dir. This covers the case where the adapter was
// just trained earlier in this same job: it is written to disk immediately,
// but isn't uploaded to R2 until the whole job finishes (see
// courier.PersistToBucket), so it must be found locally rather than
// downloaded.
func HasLocalAdapter(dir string, lang string) bool {
	filename := "adapter." + lang + ".safetensors"
	info, err := os.Stat(filepath.Join(dir, filename))
	if err != nil {
		return false
	}
	return info.Size() > 1000000 // must be GT 1Meg
}

// Check that language is supported by mms_asr, and return alternate if it is not
func CheckLanguage(ctx context.Context, lang string, sttLang string, aiTool string) (string, *log.Status) {
	var result string
	if sttLang != `` {
		result = sttLang
	} else {
		var tree = search.NewLanguageTree(ctx)
		err := tree.Load()
		if err != nil {
			return result, log.Error(ctx, 500, err, `Error loading language`)
		}
		langs, distance, err2 := tree.Search(strings.ToLower(lang), aiTool)
		if err2 != nil {
			return result, log.Error(ctx, 500, err2, `Error Searching for language`)
		}
		if len(langs) > 0 {
			result = langs[0]
			log.Info(ctx, `Using language`, result, "distance:", distance)
		} else {
			return result, log.ErrorNoErr(ctx, 400, `No compatible language code was found for`, lang)
		}
	}
	return result, nil
}
