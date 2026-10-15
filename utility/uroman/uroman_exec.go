package uroman

import (
	"context"
	"os"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/stdio_exec"
)

type UromanExec struct {
	exec     *stdio_exec.StdioExec
	useCache bool
	cache    map[string]string
}

func NewUromanExec(ctx context.Context, lang string, useCache bool) (UromanExec, *log.Status) {
	var u UromanExec
	uroman, status := stdio_exec.NewStdioExec(ctx, os.Getenv(`FCBH_MMS_ADAPTER_PYTHON`), ScriptPath(), "-l", lang)
	if status != nil {
		return u, status
	}
	u.exec = uroman
	u.useCache = useCache
	if useCache {
		u.cache = make(map[string]string)
	}
	return u, nil
}

func (u *UromanExec) Transliterate(line string) (string, *log.Status) {
	if u.useCache {
		answer, ok := u.cache[line]
		if ok {
			return answer, nil
		}
	}
	answer, status := u.exec.Process(line)
	if status != nil {
		return answer, status
	}
	if u.useCache {
		u.cache[line] = answer
	}
	return answer, nil
}
