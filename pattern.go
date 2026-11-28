package tempest

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"strings"
	"sync"
)

const maxInlinePatternParams = 8

type patternParam struct {
	name  string
	value string
}

type patternSegment struct {
	text  string
	isVar bool
	rest  bool
}

type compiledPattern[H any] struct {
	handler     H
	pattern     string
	skeleton    string
	segments    []patternSegment
	staticBytes int
	litCount    int
	hasRest     bool
}

// Concurrent exact-then-wildcard custom_id router supporting Go-style {name} and
// trailing {name...} placeholders across any delimiter scheme.
type patternRouter[H any] struct {
	exact    map[string]H
	patterns []compiledPattern[H]
	mu       sync.RWMutex
}

func newPatternRouter[H any]() *patternRouter[H] {
	return &patternRouter[H]{
		exact: make(map[string]H),
	}
}

func (pr *patternRouter[H]) register(pattern string, handler H) error {
	compiled, isWildcard, err := compilePattern(pattern, handler)
	if err != nil {
		return err
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !isWildcard {
		if _, exists := pr.exact[pattern]; exists {
			return fmt.Errorf("pattern %q is already registered", pattern)
		}
		pr.exact[pattern] = handler
		return nil
	}

	for _, existing := range pr.patterns {
		if existing.pattern == pattern || existing.skeleton == compiled.skeleton {
			return fmt.Errorf("pattern %q conflicts with already registered pattern %q", pattern, existing.pattern)
		}
	}

	pr.patterns = append(pr.patterns, compiled)
	slices.SortFunc(pr.patterns, compareCompiledPatterns[H])
	return nil
}

func (pr *patternRouter[H]) unregister(pattern string) bool {
	if pattern == "" {
		return false
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !strings.ContainsAny(pattern, "{}") {
		if _, ok := pr.exact[pattern]; !ok {
			return false
		}
		delete(pr.exact, pattern)
		return true
	}

	for i, p := range pr.patterns {
		if p.pattern == pattern {
			pr.patterns = append(pr.patterns[:i], pr.patterns[i+1:]...)
			return true
		}
	}

	return false
}

func (pr *patternRouter[H]) clear() {
	pr.mu.Lock()
	clear(pr.exact)
	pr.patterns = nil
	pr.mu.Unlock()
}

func (pr *patternRouter[H]) all() iter.Seq[string] {
	return func(yield func(string) bool) {
		pr.mu.RLock()
		keys := make([]string, 0, len(pr.exact)+len(pr.patterns))
		for k := range pr.exact {
			keys = append(keys, k)
		}
		for _, p := range pr.patterns {
			keys = append(keys, p.pattern)
		}
		pr.mu.RUnlock()

		slices.Sort(keys)
		for _, k := range keys {
			if !yield(k) {
				return
			}
		}
	}
}

func (pr *patternRouter[H]) match(customID string) (H, [maxInlinePatternParams]patternParam, []patternParam, uint8, bool) {
	var (
		zero       H
		params     [maxInlinePatternParams]patternParam
		extra      []patternParam
		paramCount uint8
	)

	pr.mu.RLock()
	defer pr.mu.RUnlock()

	if h, ok := pr.exact[customID]; ok {
		return h, params, nil, 0, true
	}

	for i := range pr.patterns {
		p := &pr.patterns[i]
		var n uint8
		extra = extra[:0]

		if matchCompiledPattern(p.segments, customID, &params, &extra, &n) {
			return p.handler, params, extra, n, true
		}
	}

	return zero, params, nil, paramCount, false
}

func compilePattern[H any](pattern string, handler H) (compiledPattern[H], bool, error) {
	if pattern == "" {
		return compiledPattern[H]{}, false, fmt.Errorf("pattern must not be empty")
	}

	if !strings.ContainsAny(pattern, "{}") {
		return compiledPattern[H]{handler: handler, pattern: pattern}, false, nil
	}

	var (
		segments    []patternSegment
		seenParams  = make(map[string]struct{})
		skeletonBuf strings.Builder
		staticBytes int
		litCount    int
		hasRest     bool
	)

	for i := 0; i < len(pattern); {
		openIdx := strings.IndexByte(pattern[i:], '{')
		if openIdx < 0 {
			lit := pattern[i:]
			if strings.ContainsRune(lit, '}') {
				return compiledPattern[H]{}, false, fmt.Errorf("unmatched '}' in pattern %q", pattern)
			}

			segments = append(segments, patternSegment{text: lit})
			skeletonBuf.WriteString(lit)
			staticBytes += len(lit)
			litCount++
			break
		}

		if openIdx > 0 {
			lit := pattern[i : i+openIdx]
			if strings.ContainsRune(lit, '}') {
				return compiledPattern[H]{}, false, fmt.Errorf("unmatched '}' in pattern %q", pattern)
			}

			segments = append(segments, patternSegment{text: lit})
			skeletonBuf.WriteString(lit)
			staticBytes += len(lit)
			litCount++
		} else if len(segments) > 0 && segments[len(segments)-1].isVar {
			return compiledPattern[H]{}, false, fmt.Errorf("adjacent wildcards without separator in pattern %q", pattern)
		}

		start := i + openIdx + 1
		closeOffset := strings.IndexByte(pattern[start:], '}')
		if closeOffset < 0 {
			return compiledPattern[H]{}, false, fmt.Errorf("unclosed '{' in pattern %q", pattern)
		}

		rawName := pattern[start : start+closeOffset]
		if strings.ContainsRune(rawName, '{') {
			return compiledPattern[H]{}, false, fmt.Errorf("nested '{' in pattern %q", pattern)
		}

		isRest := false
		if strings.HasSuffix(rawName, "...") {
			isRest = true
			rawName = strings.TrimSuffix(rawName, "...")
		}
		if rawName == "" {
			return compiledPattern[H]{}, false, fmt.Errorf("empty wildcard name in pattern %q", pattern)
		}

		if _, dup := seenParams[rawName]; dup {
			return compiledPattern[H]{}, false, fmt.Errorf("duplicate wildcard name %q in pattern %q", rawName, pattern)
		}
		seenParams[rawName] = struct{}{}

		nextIdx := start + closeOffset + 1
		if isRest && nextIdx < len(pattern) {
			return compiledPattern[H]{}, false, fmt.Errorf("catch-all wildcard {%s...} must be at the end of pattern %q", rawName, pattern)
		}

		segments = append(segments, patternSegment{
			text:  rawName,
			isVar: true,
			rest:  isRest,
		})

		if isRest {
			hasRest = true
			skeletonBuf.WriteString("\x00*")
		} else {
			skeletonBuf.WriteString("\x00?")
		}

		i = nextIdx
	}

	return compiledPattern[H]{
		handler:     handler,
		pattern:     pattern,
		skeleton:    skeletonBuf.String(),
		segments:    segments,
		staticBytes: staticBytes,
		litCount:    litCount,
		hasRest:     hasRest,
	}, true, nil
}

func compareCompiledPatterns[H any](a, b compiledPattern[H]) int {
	// Non-catch-all patterns beat trailing {name...} patterns.
	if a.hasRest != b.hasRest {
		if !a.hasRest {
			return -1
		}
		return 1
	}

	// Patterns with more static literals / static bytes are more specific.
	if c := cmp.Compare(b.litCount, a.litCount); c != 0 {
		return c
	}
	if c := cmp.Compare(b.staticBytes, a.staticBytes); c != 0 {
		return c
	}
	if c := cmp.Compare(len(b.segments), len(a.segments)); c != 0 {
		return c
	}

	// Longer leading static literal breaks ties deterministically.
	var aLead, bLead int
	if len(a.segments) > 0 && !a.segments[0].isVar {
		aLead = len(a.segments[0].text)
	}
	if len(b.segments) > 0 && !b.segments[0].isVar {
		bLead = len(b.segments[0].text)
	}
	if c := cmp.Compare(bLead, aLead); c != 0 {
		return c
	}

	return cmp.Compare(a.pattern, b.pattern)
}

func matchCompiledPattern(segments []patternSegment, s string, params *[maxInlinePatternParams]patternParam, extra *[]patternParam, count *uint8) bool {
	appendParam := func(name, val string) {
		if int(*count) < maxInlinePatternParams {
			params[*count] = patternParam{name: name, value: val}
			*count++
			return
		}
		*extra = append(*extra, patternParam{name: name, value: val})
		*count++
	}

	for i := 0; i < len(segments); i++ {
		seg := segments[i]
		if !seg.isVar {
			if !strings.HasPrefix(s, seg.text) {
				return false
			}
			s = s[len(seg.text):]
			continue
		}

		// Trailing wildcard captures remaining suffix (catch-all permits empty, normal requires non-empty).
		if i == len(segments)-1 {
			if !seg.rest && s == "" {
				return false
			}
			appendParam(seg.text, s)
			return true
		}

		// Wildcard followed by a static literal anchors on the first occurrence >= 1 byte.
		nextLit := segments[i+1].text
		idx := strings.Index(s, nextLit)
		if idx <= 0 {
			return false
		}

		appendParam(seg.text, s[:idx])
		s = s[idx+len(nextLit):]
		i++
	}

	return s == ""
}
