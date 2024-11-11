package perf

import (
	"time"

	"github.com/philippta/godbg/debug"
)

type Profiler struct {
	start time.Time
	last  time.Time
}

func Start(name ...string) *Profiler {
	if len(name) > 0 {
		debug.Logf("=================================")
		debug.Logf("Profiling: %s", name[0])
	}
	debug.Logf("=================================")
	now := time.Now()
	return &Profiler{now, now}
}

func (p *Profiler) Mark(label string) {
	now := time.Now()
	dur := now.Sub(p.last)
	debug.Logf("%-20s %12v", label, dur)
	p.last = now
}

func (p *Profiler) End() {
	dur := time.Since(p.start)
	if p.start != p.last {
		debug.Logf("---------------------------------")
	}
	debug.Logf("Total:               %12v", dur)
	debug.Logf("=================================")
}
