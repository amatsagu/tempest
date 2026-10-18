package tempest

import (
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// TuneRuntime sizes the Go runtime to the host: GOMAXPROCS from the container's CPU
// quota and GOMEMLIMIT from its memory limit, leaving headroom. Nothing in this
// library calls it, so the process is left exactly as the developer set it up; call
// it once at startup when the process runs somewhere the Go runtime cannot see the
// limits itself, such as a CPU-limited container.
//
// It returns the CPU count and the memory limit it applied, either of which is zero
// when nothing was detected.
func TuneRuntime() (cpus int, memoryLimit int64) {
	cpus = detectCPUQuota()
	if cpus < 1 {
		cpus = runtime.NumCPU()
	}

	if cpus < 1 {
		cpus = 1
	}
	runtime.GOMAXPROCS(cpus)

	memoryLimit = detectMemoryLimit()
	applyMemoryLimit(memoryLimit)

	return cpus, memoryLimit
}

// applyMemoryLimit sets GOMEMLIMIT from a memory reading, leaving headroom.
func applyMemoryLimit(memory int64) {
	if memory <= 0 {
		return
	}

	headroom := memory / 8
	headroom = clamp64(headroom, 16<<20, 512<<20)

	limit := memory - headroom
	if limit < 32<<20 {
		limit = memory
	}

	debug.SetMemoryLimit(limit)
}

// detectCPUQuota reports the container CPU quota, or zero when there is none.
func detectCPUQuota() int {
	if v := os.Getenv("GOMAXPROCS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	// cgroup v2: "max 100000" or "<quota> <period>".
	if data, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) == 2 && fields[0] != "max" {
			quota, err1 := strconv.ParseInt(fields[0], 10, 64)
			period, err2 := strconv.ParseInt(fields[1], 10, 64)
			if err1 == nil && err2 == nil && period > 0 {
				return int(ceilDiv(quota, period))
			}
		}
	}

	// cgroup v1.
	quota := readIntFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	period := readIntFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if quota > 0 && period > 0 {
		return int(ceilDiv(quota, period))
	}

	return 0
}

// detectMemoryLimit reports the container memory limit in bytes, or zero.
func detectMemoryLimit() int64 {
	if v := os.Getenv("GOMEMLIMIT"); v != "" {
		if n, ok := parseBytes(v); ok {
			return n
		}
	}

	if n := readUintFile("/sys/fs/cgroup/memory.max"); n > 0 {
		return int64(n)
	}

	if n := readUintFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); n > 0 {
		// cgroup v1 reports a huge sentinel when unlimited; ignore it.
		if n < 1<<62 {
			return int64(n)
		}
	}

	return 0
}

func readIntFile(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	n, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func readUintFile(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	n, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// parseBytes parses a GOMEMLIMIT-style size such as "512MiB" or "1GB".
func parseBytes(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "off" {
		return 0, false
	}

	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "GiB"):
		mult, s = 1<<30, strings.TrimSuffix(s, "GiB")
	case strings.HasSuffix(s, "MiB"):
		mult, s = 1<<20, strings.TrimSuffix(s, "MiB")
	case strings.HasSuffix(s, "KiB"):
		mult, s = 1<<10, strings.TrimSuffix(s, "KiB")
	case strings.HasSuffix(s, "GB"):
		mult, s = 1e9, strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult, s = 1e6, strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}

	return n * mult, true
}

func ceilDiv(a, b int64) int64 {
	if b <= 0 {
		return a
	}
	return (a + b - 1) / b
}

func clamp64(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}
	return v
}
