package proxy

import (
	"regexp"
	"sort"
	"strings"
)

// RouteTarget defines the final destination for a request.
type RouteTarget struct {
	Provider string
	Model    string
}

// ModelMapper provides high-performance model and provider mapping.
type ModelMapper struct {
	exact         map[string]RouteTarget
	prefixes      []prefixEntry
	regexes       []regexEntry
	DefaultTarget RouteTarget
}

type prefixEntry struct {
	prefix string
	target RouteTarget
}

type regexEntry struct {
	re     *regexp.Regexp
	target RouteTarget
}

// NewModelMapper initializes a mapper with standard mappings.
func NewModelMapper() *ModelMapper {
	m := &ModelMapper{
		exact: make(map[string]RouteTarget),
		DefaultTarget: RouteTarget{
			Provider: "github",
			Model:    "gpt-4.1",
		},
	}

	// Dynamic Pattern mappings (Version-independent)
	// All Claude families are currently hardcoded to map to gpt-4.1
	m.AddPattern(".*sonnet.*", RouteTarget{"github", "gpt-4.1"})
	m.AddPattern(".*haiku.*", RouteTarget{"github", "gpt-4.1"})
	m.AddPattern(".*opus.*", RouteTarget{"github", "gpt-4.1"})

	// Legacy/General Prefix mappings
	m.AddPrefix("claude-", RouteTarget{"github", "gpt-4.1"})

	return m
}

// AddExact adds a case-insensitive exact match mapping.
func (m *ModelMapper) AddExact(model string, target RouteTarget) {
	m.exact[strings.ToLower(model)] = target
}

// AddPrefix adds a prefix-based mapping.
// Longest prefixes are matched first for maximum specificity.
func (m *ModelMapper) AddPrefix(prefix string, target RouteTarget) {
	m.prefixes = append(m.prefixes, prefixEntry{
		prefix: strings.ToLower(prefix),
		target: target,
	})
	// Keep prefixes sorted by length descending to ensure longest-match priority
	sort.Slice(m.prefixes, func(i, j int) bool {
		return len(m.prefixes[i].prefix) > len(m.prefixes[j].prefix)
	})
}

// AddPattern adds a regex-based pattern mapping.
func (m *ModelMapper) AddPattern(pattern string, target RouteTarget) {
	re, err := regexp.Compile("(?i)" + pattern) // Case-insensitive
	if err != nil {
		return
	}
	m.regexes = append(m.regexes, regexEntry{re: re, target: target})
}

// Resolve identifies the target provider and model for an input model name.
func (m *ModelMapper) Resolve(model string) (RouteTarget, bool) {
	if model == "" {
		return m.DefaultTarget, false
	}

	lowered := strings.ToLower(model)

	// 1. Check exact match cache (O(1))
	if target, ok := m.exact[lowered]; ok {
		return target, true
	}

	// 2. Check regex patterns (O(M) where M is small)
	// We check patterns before prefixes to allow specific families (like haiku)
	// to override general prefixes (like claude-).
	for _, entry := range m.regexes {
		if entry.re.MatchString(model) {
			return entry.target, true
		}
	}

	// 3. Check prefix matches (O(N) where N is small, sorted by specificity)
	for _, entry := range m.prefixes {
		if strings.HasPrefix(lowered, entry.prefix) {
			return entry.target, true
		}
	}

	// 4. Passthrough if not mapped
	return RouteTarget{Model: model}, false
}
