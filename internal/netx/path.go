package netx

import (
	"fmt"
	"net/http"
	"strings"
)

// MethodPathPattern is the parsed form of a Route.Allow key. An empty Method
// matches every HTTP method.
type MethodPathPattern struct {
	Method string
	Path   string
}

// ParseMethodPathPattern parses either a legacy path-only access pattern or a
// method-qualified pattern such as "PATCH:/api/user/". Methods are uppercase
// HTTP tokens. GET also covers HEAD during matching.
func ParseMethodPathPattern(pattern string) (MethodPathPattern, error) {
	if strings.HasPrefix(pattern, "/") {
		if err := ValidatePathPattern(pattern); err != nil {
			return MethodPathPattern{}, err
		}
		return MethodPathPattern{Path: pattern}, nil
	}

	method, path, found := strings.Cut(pattern, ":")
	if !found || method == "" || !strings.HasPrefix(path, "/") {
		return MethodPathPattern{}, fmt.Errorf(
			"access pattern %q must be /path or METHOD:/path", pattern,
		)
	}
	if method != strings.ToUpper(method) || !validHTTPToken(method) {
		return MethodPathPattern{}, fmt.Errorf(
			"access pattern %q has an invalid uppercase HTTP method", pattern,
		)
	}
	if err := ValidatePathPattern(path); err != nil {
		return MethodPathPattern{}, err
	}
	return MethodPathPattern{Method: method, Path: path}, nil
}

// FormatMethodPathPattern validates and formats the persistent Route.Allow
// key. An empty method retains the backward-compatible path-only form.
func FormatMethodPathPattern(method, path string) (string, error) {
	method = strings.TrimSpace(method)
	if method == "" {
		parsed, err := ParseMethodPathPattern(path)
		if err != nil {
			return "", err
		}
		if parsed.Method != "" {
			return "", fmt.Errorf("path must not contain an HTTP method")
		}
		return parsed.Path, nil
	}
	key := method + ":" + path
	if _, err := ParseMethodPathPattern(key); err != nil {
		return "", err
	}
	return key, nil
}

// MatchMethodPathPattern reports whether method and path match pattern.
func MatchMethodPathPattern(method, path string, pattern MethodPathPattern, rootMode RootPathMatchMode) bool {
	if !methodMatches(method, pattern.Method) {
		return false
	}
	return MatchPathPattern(path, pattern.Path, rootMode)
}

// MethodMatchRank orders rules that have the same path specificity. An exact
// method beats GET's HEAD compatibility, which beats a method-less rule.
func MethodMatchRank(requestMethod, ruleMethod string) int {
	switch {
	case ruleMethod == requestMethod:
		return 2
	case requestMethod == http.MethodHead && ruleMethod == http.MethodGet:
		return 1
	case ruleMethod == "":
		return 0
	default:
		return -1
	}
}

func methodMatches(requestMethod, ruleMethod string) bool {
	return MethodMatchRank(requestMethod, ruleMethod) >= 0
}

func validHTTPToken(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		switch b := value[index]; {
		case b >= 'A' && b <= 'Z':
		case b >= '0' && b <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)):
		default:
			return false
		}
	}
	return true
}

// RootPathMatchMode defines how the root pattern ("/") is interpreted.
type RootPathMatchMode uint8

const (
	// RootPathExact makes "/" match only the root path.
	RootPathExact RootPathMatchMode = iota
	// RootPathSubtree makes "/" match every absolute request path.
	RootPathSubtree
)

// ValidatePathPattern validates the small path-pattern language shared by
// access rules and service registrations. Patterns are exact unless they end
// in "/", in which case they match that subtree. The legacy "/*" notation is
// deliberately rejected: it looks like a general wildcard but is not one.
func ValidatePathPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("path pattern is required")
	}
	if !strings.HasPrefix(pattern, "/") {
		return fmt.Errorf("path pattern %q must start with '/'", pattern)
	}
	if strings.HasSuffix(pattern, "/*") {
		return fmt.Errorf("path pattern %q must use a trailing '/' for a subtree; '/*' is not supported", pattern)
	}
	return nil
}

// MatchPathPattern reports whether path matches pattern. Except for "/",
// patterns ending in "/" match a subtree and all other patterns match
// exactly. rootMode makes the root behavior explicit for callers whose
// resource model treats it as a subtree (such as a static-file mount).
func MatchPathPattern(path, pattern string, rootMode RootPathMatchMode) bool {
	if path == "" || pattern == "" {
		return false
	}
	if pattern == "/" {
		return rootMode == RootPathSubtree && strings.HasPrefix(path, "/") || path == "/"
	}
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}
	return path == pattern
}
