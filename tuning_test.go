package tempest

import (
	"runtime"
	"runtime/debug"
	"testing"
)

// TestNewLeaves RuntimeAlone proves building a client does not touch process-wide
// runtime settings; host tuning is opt-in through TuneRuntime.
func TestNewLeavesRuntimeAlone(t *testing.T) {
	cpus := runtime.GOMAXPROCS(0)
	limit := debug.SetMemoryLimit(-1)

	if _, err := New(Config{Token: "x"}); err != nil {
		t.Fatalf("New: %v", err)
	}

	if got := runtime.GOMAXPROCS(0); got != cpus {
		t.Errorf("GOMAXPROCS = %d after New, want %d", got, cpus)
	}

	if got := debug.SetMemoryLimit(-1); got != limit {
		t.Errorf("GOMEMLIMIT = %d after New, want %d", got, limit)
	}
}

// TestTuneRuntime proves the opt-in helper applies a CPU count and restores the
// process afterwards.
func TestTuneRuntime(t *testing.T) {
	cpus := runtime.GOMAXPROCS(0)
	limit := debug.SetMemoryLimit(-1)

	t.Cleanup(func() {
		runtime.GOMAXPROCS(cpus)
		debug.SetMemoryLimit(limit)
	})

	detected, memory := TuneRuntime()
	if detected < 1 {
		t.Fatalf("TuneRuntime reported %d CPUs, want at least 1", detected)
	}

	if memory < 0 {
		t.Fatalf("TuneRuntime reported a negative memory limit: %d", memory)
	}

	if got := runtime.GOMAXPROCS(0); got != detected {
		t.Fatalf("GOMAXPROCS = %d after TuneRuntime, want %d", got, detected)
	}
}

// TestApplyMemoryLimitLeavesHeadroom proves a detected limit is applied below the
// real ceiling, with the process setting restored.
func TestApplyMemoryLimitLeavesHeadroom(t *testing.T) {
	limit := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(limit) })

	applyMemoryLimit(0)
	if got := debug.SetMemoryLimit(-1); got != limit {
		t.Fatalf("applyMemoryLimit(0) changed the limit to %d", got)
	}

	applyMemoryLimit(1 << 30)
	if got := debug.SetMemoryLimit(-1); got >= 1<<30 {
		t.Fatalf("GOMEMLIMIT = %d, want headroom below %d", got, 1<<30)
	}
}
