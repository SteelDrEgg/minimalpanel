package web

import (
	"net/http"

	"arupa/internal/auth"
	"arupa/internal/netx"
)

// StartKernel registers host-level kernel information and control endpoints.
// Both endpoints require authentication; Route.Allow can further restrict
// them by method and path.
func StartKernel(mux *http.ServeMux, version string, reload func() error) {
	mux.HandleFunc("GET /api/kernel/version", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		_ = netx.WriteSuccess(w, "Kernel version fetched", map[string]string{
			"version": version,
		})
	}))

	mux.HandleFunc("POST /api/kernel/reload", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if reload == nil {
			_ = netx.WriteInternalServerError(w, "Configuration reload is unavailable", nil)
			return
		}
		if err := reload(); err != nil {
			_ = netx.WriteInternalServerError(w, "Failed to reload configuration", err)
			return
		}

		_ = netx.WriteSuccess(w, "Configuration reloaded", nil)
	}))
}
