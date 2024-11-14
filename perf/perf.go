package perf

import (
	"time"

	"github.com/philippta/godbg/debug"
)

const disabled = true

type Profiler struct {
	start time.Time
	last  time.Time
}

func Start(name ...string) *Profiler {
	if disabled {
		return nil
	}
	if len(name) > 0 {
		debug.Logf("=================================")
		debug.Logf("Profiling: %s", name[0])
	}
	debug.Logf("=================================")
	now := time.Now()
	return &Profiler{now, now}
}

func (p *Profiler) Mark(label string) {
	if p == nil {
		return
	}
	now := time.Now()
	dur := now.Sub(p.last)
	debug.Logf("%-20s %12v", label, dur)
	p.last = now
}

func (p *Profiler) End() {
	if p == nil {
		return
	}
	dur := time.Since(p.start)
	if p.start != p.last {
		debug.Logf("---------------------------------")
	}
	debug.Logf("Total:               %12v", dur)
	debug.Logf("=================================")
}
