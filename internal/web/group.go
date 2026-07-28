package web

import (
	"net/http"
	"sort"
	"strings"

	"arupa/internal/conf"
	"arupa/internal/netx"
)

type groupView struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
}

type groupPutRequest struct {
	Users *[]string `json:"users"`
}

func startGroupConfig(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/group", management(conf.APICapabilityGroup, handleGroups))
	mux.HandleFunc("GET /api/group/{name}", management(conf.APICapabilityGroup, handleGroup))
	mux.HandleFunc("PUT /api/group/{name}", management(conf.APICapabilityGroup, handlePutGroup))
	mux.HandleFunc("DELETE /api/group/{name}", management(conf.APICapabilityGroup, handleDeleteGroup))
}

func handleGroups(w http.ResponseWriter, _ *http.Request) {
	configured := conf.GetGroups()
	names := make([]string, 0, len(configured))
	for name := range configured {
		names = append(names, name)
	}
	sort.Strings(names)

	groups := make([]groupView, 0, len(names))
	for _, name := range names {
		groups = append(groups, groupView{Name: name, Users: configured[name]})
	}
	_ = netx.WriteSuccess(w, "Groups fetched", map[string]any{"groups": groups})
}

func handleGroup(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	users, ok := conf.GetGroups()[name]
	if !ok {
		_ = netx.WriteNotFound(w)
		return
	}
	_ = netx.WriteSuccess(w, "Group fetched", groupView{Name: name, Users: users})
}

func handlePutGroup(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		_ = netx.WriteBadRequest(w, "Group name is required")
		return
	}
	var request groupPutRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	if request.Users == nil {
		_ = netx.WriteBadRequest(w, "users is required and cannot be null")
		return
	}
	if err := conf.Update(conf.Set(
		conf.JoinPath(string(conf.ConfigFieldGroups), name),
		append([]string(nil), (*request.Users)...),
	)); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update group", err)
		return
	}
	_ = netx.WriteSuccess(w, "Group updated", groupView{Name: name, Users: *request.Users})
}

func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		_ = netx.WriteBadRequest(w, "Group name is required")
		return
	}
	if err := conf.Update(conf.Remove(conf.JoinPath(string(conf.ConfigFieldGroups), name))); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to delete group", err)
		return
	}
	_ = netx.WriteSuccess(w, "Group deleted", map[string]string{"name": name})
}
