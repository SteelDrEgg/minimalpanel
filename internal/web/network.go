package web

import (
	"net/http"
	"strings"

	"arupa/internal/conf"
	"arupa/internal/netx"
)

type listenRequest struct {
	Listen string `json:"listen"`
}

type tlsRequest struct {
	TLS *bool `json:"tls"`
}

func startNetworkConfig(mux *http.ServeMux, applied AppliedConfig) {
	mux.HandleFunc("GET /api/network/listen", management(conf.APICapabilityNetwork, func(w http.ResponseWriter, _ *http.Request) {
		writeListen(w, applied.Listen, "Listen address fetched")
	}))
	mux.HandleFunc("PATCH /api/network/listen", management(conf.APICapabilityNetwork, func(w http.ResponseWriter, r *http.Request) {
		var request listenRequest
		if !decodeRequest(w, r, &request) {
			return
		}
		request.Listen = strings.TrimSpace(request.Listen)
		if request.Listen == "" {
			_ = netx.WriteBadRequest(w, "listen is required")
			return
		}
		if err := conf.Update(conf.Set(conf.JoinPath(string(conf.ConfigFieldListen)), request.Listen)); err != nil {
			_ = netx.WriteInternalServerError(w, "Failed to update listen address", err)
			return
		}
		writeListen(w, applied.Listen, "Listen address updated")
	}))

	mux.HandleFunc("GET /api/network/tls", management(conf.APICapabilityNetwork, func(w http.ResponseWriter, _ *http.Request) {
		writeTLS(w, applied.TLS, "TLS setting fetched")
	}))
	mux.HandleFunc("PATCH /api/network/tls", management(conf.APICapabilityNetwork, func(w http.ResponseWriter, r *http.Request) {
		var request tlsRequest
		if !decodeRequest(w, r, &request) {
			return
		}
		if request.TLS == nil {
			_ = netx.WriteBadRequest(w, "tls is required and cannot be null")
			return
		}
		if err := conf.Update(conf.Set(conf.JoinPath(string(conf.ConfigFieldTLS)), *request.TLS)); err != nil {
			_ = netx.WriteInternalServerError(w, "Failed to update TLS", err)
			return
		}
		writeTLS(w, applied.TLS, "TLS setting updated")
	}))
}

func writeListen(w http.ResponseWriter, applied, message string) {
	current := conf.GetListen()
	_ = netx.WriteSuccess(w, message, map[string]any{
		"listen":           current,
		"requires_restart": current != applied,
	})
}

func writeTLS(w http.ResponseWriter, applied bool, message string) {
	current := conf.GetTLS()
	_ = netx.WriteSuccess(w, message, map[string]any{
		"tls":              current,
		"requires_restart": current != applied,
	})
}
