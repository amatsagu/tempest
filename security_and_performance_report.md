# Codebase Security and Performance Report

This document outlines the findings from a recent analysis of the codebase, focusing on security vulnerabilities and performance pitfalls.

## 1. Security & Reliability: Runtime Panics in Helper Methods

**Location:** `mentionable.go` (Methods: `GuildAvatarURL`, `GuildBannerURL`)

**Finding:**
The library uses `panic` in helper methods when an expected state is not met. Specifically, if a `Member` struct is missing the `GuildID` (which is equal to 0), calling `GuildAvatarURL` or `GuildBannerURL` will panic and crash the application.

**Code Example:**
```go
func (member Member) GuildAvatarURL() string {
	if member.GuildAvatarHash == "" {
		return ""
	}

	if member.GuildID == 0 {
		panic("member struct is missing guild ID which is required in avatar url method - it appears to be problem of your custom tempest client implementation")
	}
    // ...
}
```

**Impact/Abuse:**
While this is intended to catch errors in custom client implementations, a `panic` in a library meant to be embedded in long-running services (like a Discord bot) is highly risky. If an edge case or a bug in Discord's API payload results in a missing `GuildID`, and the bot attempts to log or use the avatar URL, the entire bot process will crash. This acts as a potential self-inflicted Denial of Service (DoS) vulnerability.

**Suggestion:**
Remove the `panic`. Instead, handle the missing state gracefully.
- **Option 1:** Return an empty string or a default value, optionally logging a warning.
- **Option 2:** Change the method signature to return an error (`func (member Member) GuildAvatarURL() (string, error)`) so the caller can handle it, though this is a breaking change.

Returning an empty string is usually the safest non-breaking approach for URL getters:
```go
	if member.GuildID == 0 {
		return "" // Or log an internal error, but do not panic.
	}
```

---

## 2. Performance: Lock Contention in SharedMap

**Location:** `shared-map.go` (Methods: `ReadRange`, `FilterMap`, `FilterValues`, `FilterKeys`, `Sweep`)

**Finding:**
The `SharedMap` structure uses `sync.RWMutex` to ensure thread safety. However, methods that iterate over the entire map (`ReadRange`, `FilterMap`, etc.) hold the Read Lock (`RLock`) or Write Lock (`Lock` in `Sweep`) for the *entire duration* of the iteration and the execution of the user-provided callback function `fn`.

**Code Example:**
```go
func (sm *SharedMap[K, V]) ReadRange(fn func(key K, value V)) {
	sm.mu.RLock()
	for key, value := range sm.cache {
		fn(key, value) // User provided function runs while holding the lock
	}
	sm.mu.RUnlock()
}
```

**Impact/Abuse:**
If a developer passes a callback function that performs blocking operations (like I/O, network requests, or heavy computation), or if the map simply contains a massive number of elements, the RWMutex will remain locked. This will block all other goroutines attempting to write to the map (and potentially reads depending on the RWMutex implementation and lock type), causing severe performance bottlenecks and increased latency across the application.

**Suggestion:**
- The current implementation has a comment warning users not to perform heavy computations. While good, it relies entirely on the developer reading and following the comment.
- **Alternative Approach:** If iteration with heavy callbacks is a common use case, consider allowing a snapshot approach where keys or key-value pairs are copied to a slice under the lock, and then the slice is iterated over *outside* the lock.

```go
func (sm *SharedMap[K, V]) Snapshot() map[K]V {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	snap := make(map[K]V, len(sm.cache))
	for k, v := range sm.cache {
		snap[k] = v
	}
	return snap
}
```
Then the user can iterate over the snapshot safely without blocking the main map.

---

## 3. Performance: Memory Allocations in HTTP Client Verification

**Location:** `http-client.go` (Function: `verifyRequest`)

**Finding:**
In the HTTP request verification process, `sync.Pool` is used efficiently to minimize buffer allocations when reading the request body. However, after successful verification, the body is copied into a newly allocated byte slice before the buffer is returned to the pool.

**Code Example:**
```go
	if ed25519.Verify(key, buf.Bytes(), sig) {
		// We copy the body bytes because we're returning the buffer to the pool.
		bodyBytes := make([]byte, buf.Len()-len(timestamp))
		copy(bodyBytes, buf.Bytes()[len(timestamp):])
		return bodyBytes, true
	}
```

**Impact/Abuse:**
For a high-traffic Discord bot receiving hundreds or thousands of interactions per second, creating a new `[]byte` slice (`make([]byte, ...)`) for every single valid request creates significant garbage collection (GC) pressure. While `sync.Pool` saves on the initial read, the subsequent `make` negates some of those performance gains.

**Suggestion:**
This is a common trade-off in Go. Returning the buffer to the pool requires copying the data if the caller needs it later. To truly optimize this:
1.  **Pass the Buffer:** If the downstream handler can process the data synchronously, pass a `[]byte` slice that is backed by the pooled buffer, and only return the buffer to the pool *after* the handler finishes. This requires changing the architecture to allow deferred pool returns.
2.  If the current architecture must return a copied `[]byte` because the handler might spin up a goroutine and keep the bytes around, the current implementation is acceptable, but the performance implications under high load should be documented.

---

## 4. Security: Request Verification (Positive Finding)

**Location:** `http-client.go`, `constant.go`

**Finding:**
The implementation correctly utilizes Ed25519 signatures to verify incoming requests from Discord, which is a mandatory security requirement. Furthermore, it implements a hard limit on request body sizes (`MAX_REQUEST_BODY_SIZE` = 1MB) using `io.LimitReader`.

**Impact/Abuse:**
This effectively mitigates basic Denial of Service (DoS) attacks where a malicious actor might try to send massive payloads to exhaust server memory or bandwidth on the interactions endpoint.

**Suggestion:**
Maintain this pattern. No changes needed here, it is well implemented.
