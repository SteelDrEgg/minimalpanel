package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"arupa/internal/conf"
	"arupa/internal/netx"
)

type pageView struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type pageRequest struct {
	Path string `json:"path"`
}

func startPageConfig(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/page", management(conf.APICapabilityPages, handlePages))
	mux.HandleFunc("PUT /api/page/{status}", management(conf.APICapabilityPages, handlePutPage))
	mux.HandleFunc("DELETE /api/page/{status}", management(conf.APICapabilityPages, handleDeletePage))
}

func handlePages(w http.ResponseWriter, _ *http.Request) {
	configured := conf.GetPages()
	statuses := make([]string, 0, len(configured))
	for status := range configured {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	pages := make([]pageView, 0, len(statuses))
	for _, status := range statuses {
		pages = append(pages, pageView{Status: status, Path: configured[status]})
	}
	_ = netx.WriteSuccess(w, "Pages fetched", map[string]any{"pages": pages})
}

func handlePutPage(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.PathValue("status"))
	if !validHTTPStatus(status) {
		_ = netx.WriteBadRequest(w, "Status must be an HTTP status code from 100 to 599")
		return
	}
	var request pageRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	request.Path = strings.TrimSpace(request.Path)
	if request.Path == "" || !strings.HasPrefix(request.Path, "/") || strings.HasPrefix(request.Path, "//") {
		_ = netx.WriteBadRequest(w, "Page path must be a local absolute path")
		return
	}
	if err := conf.Update(conf.Set(
		conf.JoinPath(string(conf.ConfigFieldPages), status), request.Path,
	)); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update page", err)
		return
	}
	_ = netx.WriteSuccess(w, "Page updated", pageView{Status: status, Path: request.Path})
}

func handleDeletePage(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.PathValue("status"))
	if !validHTTPStatus(status) {
		_ = netx.WriteBadRequest(w, "Status must be an HTTP status code from 100 to 599")
		return
	}
	if err := conf.Update(conf.Remove(conf.JoinPath(string(conf.ConfigFieldPages), status))); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to delete page", err)
		return
	}
	_ = netx.WriteSuccess(w, "Page deleted", map[string]string{"status": status})
}

func validHTTPStatus(raw string) bool {
	status, err := strconv.Atoi(raw)
	return err == nil && status >= 100 && status <= 599
}
