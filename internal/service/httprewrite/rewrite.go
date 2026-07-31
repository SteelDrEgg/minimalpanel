// Package httprewrite applies route-relative HTTP request and response
// rewriting independently of the transport used to serve a route.
package httprewrite

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"arupa/internal/service/spec"
)

const ForwardedPrefixHeader = "X-Forwarded-Prefix"

type contextKey struct{}

type state struct {
	prefix   string
	location bool
}

type responseWriter struct {
	http.ResponseWriter
	request     *http.Request
	wroteHeader bool
}

// PrefixEnabled reports the effective prefix behavior. HTTP routes strip their
// external mount prefix by default; an explicit false value preserves it.
func PrefixEnabled(rule *spec.RewriteRule) bool {
	return rule == nil || rule.Prefix == nil || *rule.Prefix
}

// LocationEnabled reports the effective Location behavior. Root-relative
// Location headers are preserved unless explicitly enabled.
func LocationEnabled(rule *spec.RewriteRule) bool {
	return rule != nil && rule.Location != nil && *rule.Location
}

// Handler wraps next with the route's request rewrite. Matching and access
// checks must run before this handler so they continue to use the external path.
func Handler(pattern string, rule *spec.RewriteRule, next http.Handler) http.Handler {
	if next == nil || !PrefixEnabled(rule) {
		return next
	}
	prefix := strings.TrimSuffix(pattern, "/")
	if prefix == "" {
		return next
	}
	location := LocationEnabled(rule)
	stripped := http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "" {
			clone := request.Clone(request.Context())
			clone.URL.Path = "/"
			clone.URL.RawPath = ""
			request = clone
		}
		next.ServeHTTP(w, request)
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		rewrite := state{prefix: prefix, location: location}
		ctx := context.WithValue(request.Context(), contextKey{}, rewrite)
		stripped.ServeHTTP(w, request.WithContext(ctx))
	})
}

// SetForwardedPrefix replaces any caller-supplied forwarding prefix with the
// prefix established by the matched route.
func SetForwardedPrefix(headers http.Header, request *http.Request) {
	if headers == nil {
		return
	}
	headers.Del(ForwardedPrefixHeader)
	if rewrite, ok := fromRequest(request); ok {
		headers.Set(ForwardedPrefixHeader, rewrite.prefix)
	}
}

// RewriteResponseHeaders applies route-relative response rewriting in place.
func RewriteResponseHeaders(headers http.Header, request *http.Request) {
	rewrite, ok := fromRequest(request)
	if !ok || !rewrite.location || headers == nil {
		return
	}
	locations := headers.Values("Location")
	if len(locations) == 0 {
		return
	}
	headers.Del("Location")
	for _, location := range locations {
		headers.Add("Location", RewriteLocation(location, rewrite.prefix))
	}
}

// ResponseWriter rewrites response headers when the wrapped handler commits
// them. It is intended for non-streaming handlers such as static file serving.
func ResponseWriter(w http.ResponseWriter, request *http.Request) http.ResponseWriter {
	rewrite, ok := fromRequest(request)
	if !ok || !rewrite.location {
		return w
	}
	return &responseWriter{ResponseWriter: w, request: request}
}

// RewriteLocation prepends prefix to a root-relative Location. Relative,
// scheme-relative, absolute, and invalid locations are returned unchanged.
func RewriteLocation(location, prefix string) string {
	if !strings.HasPrefix(location, "/") || strings.HasPrefix(location, "//") {
		return location
	}
	reference, err := url.Parse(location)
	if err != nil || reference.IsAbs() || reference.Host != "" {
		return location
	}
	reference.Path = prefix + reference.Path
	if reference.RawPath != "" {
		reference.RawPath = (&url.URL{Path: prefix}).EscapedPath() + reference.RawPath
	}
	return reference.String()
}

func fromRequest(request *http.Request) (state, bool) {
	if request == nil {
		return state{}, false
	}
	rewrite, ok := request.Context().Value(contextKey{}).(state)
	return rewrite, ok && rewrite.prefix != ""
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	RewriteResponseHeaders(w.Header(), w.request)
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
