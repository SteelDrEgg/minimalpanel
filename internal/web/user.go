package web

import (
	"net/http"
	"sort"
	"strings"

	"arupa/internal/auth"
	"arupa/internal/conf"
	"arupa/internal/netx"
)

type userView struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
}

type userPasswordRequest struct {
	Password string `json:"password"`
}

func startUserConfig(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/user", management(conf.APICapabilityUser, handleUsers))
	mux.HandleFunc("GET /api/user/{name}", management(conf.APICapabilityUser, handleUser))
	mux.HandleFunc("PUT /api/user/{name}", management(conf.APICapabilityUser, handlePutUser))
	mux.HandleFunc("DELETE /api/user/{name}", management(conf.APICapabilityUser, handleDeleteUser))
}

func handleUsers(w http.ResponseWriter, _ *http.Request) {
	memberships := conf.GetUsersWithGroups()
	names := make([]string, 0, len(memberships))
	for name := range memberships {
		names = append(names, name)
	}
	sort.Strings(names)

	users := make([]userView, 0, len(names))
	for _, name := range names {
		users = append(users, userView{Name: name, Groups: nonNilStrings(memberships[name])})
	}
	_ = netx.WriteSuccess(w, "Users fetched", map[string]any{"users": users})
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	groups, ok := conf.GetUserWithGroups(name)
	if !ok {
		_ = netx.WriteNotFound(w)
		return
	}
	_ = netx.WriteSuccess(w, "User fetched", userView{Name: name, Groups: nonNilStrings(groups)})
}

func handlePutUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		_ = netx.WriteBadRequest(w, "Username is required")
		return
	}
	var request userPasswordRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	if request.Password == "" {
		_ = netx.WriteBadRequest(w, "Password is required")
		return
	}
	if err := auth.NewUser(name, request.Password); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update user", err)
		return
	}
	groups, _ := conf.GetUserWithGroups(name)
	_ = netx.WriteSuccess(w, "User updated", userView{Name: name, Groups: nonNilStrings(groups)})
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		_ = netx.WriteBadRequest(w, "Username is required")
		return
	}
	if err := conf.Update(conf.Remove(conf.JoinPath(string(conf.ConfigFieldUsers), name))); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to delete user", err)
		return
	}
	_ = netx.WriteSuccess(w, "User deleted", map[string]string{"name": name})
}
