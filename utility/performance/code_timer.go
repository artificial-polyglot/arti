package performance

import (
	"fmt"
	"time"
)

type CodeTimer struct {
	start time.Time
}

func NewCodeTimer() CodeTimer {
	return CodeTimer{start: time.Now()}
}

func (t *CodeTimer) Duration(step string) {
	elapsed := time.Since(t.start)
	fmt.Printf("%s Elapsed: %v\n", step, elapsed)
	t.start = time.Now()
}

func (t *CodeTimer) DurationMS(step string) {
	elapsed := time.Since(t.start)
	fmt.Printf("%s Elapsed: %.3fms\n", step, float64(elapsed)/float64(time.Millisecond))
	t.start = time.Now()
}
