package web

import (
	"net/http"

	"arupa/internal/conf"
)

// AppliedConfig records the immutable configuration values used to construct
// the current kernel process. They are runtime facts, not another config
// source, and are used only to derive requires_restart.
type AppliedConfig struct {
	Listen string
	TLS    bool
	Log    conf.LogConfig
}

// StartConfig registers configuration management endpoints other than Service.
func StartConfig(mux *http.ServeMux, applied AppliedConfig) {
	startUserConfig(mux)
	startGroupConfig(mux)
	startPageConfig(mux)
	startAccessConfig(mux)
	startLogConfig(mux, applied.Log)
	startNetworkConfig(mux, applied)
}
