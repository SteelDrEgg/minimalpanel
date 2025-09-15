package web

import (
	"minimalpanel/internal/auth"
	"minimalpanel/internal/conf"
	"minimalpanel/internal/netx"
	"net/http"
	"path/filepath"
)

// StartIndex registers index/dashboard routes with the given mux
func StartIndex(mux *http.ServeMux) {
	mux.HandleFunc("/", auth.RequireAuth(handleIndex))
}

// handleIndex serves the main dashboard page
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		netx.WriteMethodNotAllowed(w)
		return
	}

	http.ServeFile(w, r, filepath.Join(conf.GetWeb().RootPath, "index.html"))
}
