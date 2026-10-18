// Package route is a small, allocation-conscious pattern router in the spirit of
// net/http's ServeMux: static segments and {name} wildcards, separated by '/' or ':',
// generic over the handler type. Matching writes captures into a caller-owned Params.
package route

import "strings"

// MaxParams is the number of wildcard captures a single route may have.
const MaxParams = 8

// Params holds wildcard captures for one match; its zero value is ready to use.
type Params struct {
	keys   [MaxParams]string
	values [MaxParams]string
	n      int
}

func (p *Params) reset() { p.n = 0 }

func (p *Params) add(key, value string) bool {
	if p.n >= MaxParams {
		return false
	}
	p.keys[p.n] = key
	p.values[p.n] = value
	p.n++
	return true
}

// Len returns the number of captured parameters.
func (p *Params) Len() int { return p.n }

// Get returns the value captured for the named wildcard.
func (p *Params) Get(name string) (string, bool) {
	for i := 0; i < p.n; i++ {
		if p.keys[i] == name {
			return p.values[i], true
		}
	}
	return "", false
}

// At returns the i-th captured key and value.
func (p *Params) At(i int) (key, value string) {
	if i < 0 || i >= p.n {
		return "", ""
	}
	return p.keys[i], p.values[i]
}

// Router matches request paths to typed handlers.
type Router[T any] struct {
	root node[T]
}

type node[T any] struct {
	handler   T
	children  map[string]*node[T]
	wildcard  *node[T]
	pattern   string
	paramName string
	hasValue  bool
}

// Handle registers a handler for a pattern, where a "{id}" segment captures one
// segment. A leading '/' is optional; it returns an error on an invalid or duplicate
// pattern.
func (r *Router[T]) Handle(pattern string, handler T) error {
	if pattern == "" {
		return errInvalidPattern
	}
	original := pattern
	if pattern[0] != '/' {
		pattern = "/" + pattern
	}

	n := &r.root
	start := 1
	for i := 1; i <= len(pattern); i++ {
		if i != len(pattern) && pattern[i] != '/' && pattern[i] != ':' {
			continue
		}

		seg := pattern[start:i]
		start = i + 1

		if seg == "" {
			continue
		}

		if len(seg) >= 2 && seg[0] == '{' && seg[len(seg)-1] == '}' {
			name := seg[1 : len(seg)-1]
			if name == "" {
				return errInvalidPattern
			}

			if n.wildcard == nil {
				n.wildcard = &node[T]{}
			} else if n.wildcard.paramName != name {
				return errConflict
			}
			n.wildcard.paramName = name
			n = n.wildcard
			continue
		}

		if n.children == nil {
			n.children = make(map[string]*node[T])
		}
		child := n.children[seg]
		if child == nil {
			child = &node[T]{}
			n.children[seg] = child
		}
		n = child
	}

	if n.hasValue {
		return errConflict
	}
	n.handler = handler
	n.pattern = original
	n.hasValue = true
	return nil
}

// Match resolves path to a handler and its parameters. params must not be nil; its
// previous contents are overwritten.
func (r *Router[T]) Match(path string, params *Params) (T, bool) {
	handler, _, ok := r.MatchRoute(path, params)
	return handler, ok
}

// MatchRoute is Match plus the registered pattern that matched, for diagnostics.
func (r *Router[T]) MatchRoute(path string, params *Params) (T, string, bool) {
	params.reset()

	n := &r.root
	start := 0

	for i := 0; i <= len(path); i++ {
		if i != len(path) && path[i] != '/' && path[i] != ':' {
			continue
		}

		seg := path[start:i]
		start = i + 1

		if seg == "" {
			continue
		}

		if child := n.children[seg]; child != nil {
			n = child
			continue
		}

		if n.wildcard == nil {
			var zero T
			return zero, "", false
		}

		if !params.add(n.wildcard.paramName, seg) {
			var zero T
			return zero, "", false
		}
		n = n.wildcard
	}

	if !n.hasValue {
		var zero T
		return zero, "", false
	}
	return n.handler, n.pattern, true
}

// Remove unregisters the handler for pattern, pruning empty nodes, and reports
// whether one was removed. It is not synchronized with Match or Handle; callers
// sharing a Router across goroutines must provide their own locking.
func (r *Router[T]) Remove(pattern string) bool {
	if pattern == "" {
		return false
	}

	segs := PathSegments(pattern)
	if len(segs) == 0 {
		return false
	}
	return removeSegs(&r.root, segs)
}

func removeSegs[T any](n *node[T], segs []string) bool {
	seg := segs[0]
	last := len(segs) == 1

	if len(seg) >= 2 && seg[0] == '{' && seg[len(seg)-1] == '}' {
		// Only remove when the wildcard name matches; a different name at the same
		// position is a different pattern.
		if n.wildcard == nil || n.wildcard.paramName != seg[1:len(seg)-1] {
			return false
		}

		if last {
			if !n.wildcard.hasValue {
				return false
			}
			var zero T
			n.wildcard.handler = zero
			n.wildcard.hasValue = false
			if n.wildcard.empty() {
				n.wildcard = nil
			}
			return true
		}

		if !removeSegs(n.wildcard, segs[1:]) {
			return false
		}

		if n.wildcard.empty() {
			n.wildcard = nil
		}
		return true
	}

	child := n.children[seg]
	if child == nil {
		return false
	}

	if last {
		if !child.hasValue {
			return false
		}
		var zero T
		child.handler = zero
		child.hasValue = false
		if child.empty() {
			delete(n.children, seg)
		}
		return true
	}

	if !removeSegs(child, segs[1:]) {
		return false
	}

	if child.empty() {
		delete(n.children, seg)
	}
	return true
}

func (n *node[T]) empty() bool {
	return !n.hasValue && n.wildcard == nil && len(n.children) == 0
}

// PathSegments splits a pattern/path into segments using '/' and ':'.
func PathSegments(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == ':' })
}
