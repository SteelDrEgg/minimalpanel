package web

import (
	"net/http"
	"sort"
	"strings"

	"arupa/internal/conf"
	"arupa/internal/netx"
)

type accessRuleRequest struct {
	Method string    `json:"method,omitempty"`
	Path   string    `json:"path"`
	Groups *[]string `json:"groups,omitempty"`
}

type accessRuleView struct {
	Method string   `json:"method,omitempty"`
	Path   string   `json:"path"`
	Groups []string `json:"groups"`
}

func startAccessConfig(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/access", management(conf.APICapabilityAccess, handleAccess))
	mux.HandleFunc("PUT /api/access", management(conf.APICapabilityAccess, handlePutAccess))
	mux.HandleFunc("DELETE /api/access", management(conf.APICapabilityAccess, handleDeleteAccess))
}

func handleAccess(w http.ResponseWriter, _ *http.Request) {
	allow := conf.GetRouteAllow()
	rules := make([]accessRuleView, 0, len(allow))
	for key, groups := range allow {
		pattern, err := netx.ParseMethodPathPattern(key)
		if err != nil {
			continue
		}
		rules = append(rules, accessRuleView{
			Method: pattern.Method,
			Path:   pattern.Path,
			Groups: groups,
		})
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Path == rules[j].Path {
			return rules[i].Method < rules[j].Method
		}
		return rules[i].Path < rules[j].Path
	})
	_ = netx.WriteSuccess(w, "Access rules fetched", map[string]any{"rules": rules})
}

func handlePutAccess(w http.ResponseWriter, r *http.Request) {
	var request accessRuleRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	if request.Groups == nil {
		_ = netx.WriteBadRequest(w, "groups is required and cannot be null")
		return
	}
	key, ok := accessRuleKey(w, request)
	if !ok {
		return
	}
	if err := conf.Update(conf.Set(
		conf.JoinPath(string(conf.ConfigFieldRoute), string(conf.RouteFieldAllow), key),
		append([]string(nil), (*request.Groups)...),
	)); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update access rule", err)
		return
	}
	pattern, _ := netx.ParseMethodPathPattern(key)
	_ = netx.WriteSuccess(w, "Access rule updated", accessRuleView{
		Method: pattern.Method, Path: pattern.Path, Groups: *request.Groups,
	})
}

func handleDeleteAccess(w http.ResponseWriter, r *http.Request) {
	var request accessRuleRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	key, ok := accessRuleKey(w, request)
	if !ok {
		return
	}
	if err := conf.Update(conf.Remove(
		conf.JoinPath(string(conf.ConfigFieldRoute), string(conf.RouteFieldAllow), key),
	)); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to delete access rule", err)
		return
	}
	pattern, _ := netx.ParseMethodPathPattern(key)
	_ = netx.WriteSuccess(w, "Access rule deleted", map[string]string{
		"method": pattern.Method,
		"path":   pattern.Path,
	})
}

func accessRuleKey(w http.ResponseWriter, request accessRuleRequest) (string, bool) {
	request.Method = strings.TrimSpace(request.Method)
	request.Path = strings.TrimSpace(request.Path)
	key, err := netx.FormatMethodPathPattern(request.Method, request.Path)
	if err != nil {
		_ = netx.WriteBadRequest(w, err.Error())
		return "", false
	}
	return key, true
}
