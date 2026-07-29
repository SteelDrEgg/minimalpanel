package web

import (
	"net/http"
	"strings"

	"arupa/internal/conf"
	"arupa/internal/netx"
)

type logLevelRequest struct {
	Level string `json:"level"`
}

type logFormatRequest struct {
	Format string `json:"format"`
}

func startLogConfig(mux *http.ServeMux, applied conf.LogConfig) {
	mux.HandleFunc("GET /api/log/level", management(conf.APICapabilityLog, func(w http.ResponseWriter, _ *http.Request) {
		writeLogLevel(w, applied.Level, "Log level fetched")
	}))
	mux.HandleFunc("PATCH /api/log/level", management(conf.APICapabilityLog, func(w http.ResponseWriter, r *http.Request) {
		var request logLevelRequest
		if !decodeRequest(w, r, &request) {
			return
		}
		request.Level = strings.TrimSpace(request.Level)
		if request.Level == "" {
			_ = netx.WriteBadRequest(w, "level is required")
			return
		}
		if err := conf.Update(conf.Set(
			conf.JoinPath(string(conf.ConfigFieldLog), string(conf.LogFieldLevel)),
			request.Level,
		)); err != nil {
			_ = netx.WriteInternalServerError(w, "Failed to update log level", err)
			return
		}
		writeLogLevel(w, applied.Level, "Log level updated")
	}))

	mux.HandleFunc("GET /api/log/format", management(conf.APICapabilityLog, func(w http.ResponseWriter, _ *http.Request) {
		writeLogFormat(w, applied.Format, "Log format fetched")
	}))
	mux.HandleFunc("PATCH /api/log/format", management(conf.APICapabilityLog, func(w http.ResponseWriter, r *http.Request) {
		var request logFormatRequest
		if !decodeRequest(w, r, &request) {
			return
		}
		request.Format = strings.TrimSpace(request.Format)
		if request.Format == "" {
			_ = netx.WriteBadRequest(w, "format is required")
			return
		}
		if err := conf.Update(conf.Set(
			conf.JoinPath(string(conf.ConfigFieldLog), string(conf.LogFieldFormat)),
			request.Format,
		)); err != nil {
			_ = netx.WriteInternalServerError(w, "Failed to update log format", err)
			return
		}
		writeLogFormat(w, applied.Format, "Log format updated")
	}))
}

func writeLogLevel(w http.ResponseWriter, applied, message string) {
	current := conf.GetLog().Level
	_ = netx.WriteSuccess(w, message, map[string]any{
		"level":            current,
		"requires_restart": !strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(applied)),
	})
}

func writeLogFormat(w http.ResponseWriter, applied, message string) {
	current := conf.GetLog().Format
	_ = netx.WriteSuccess(w, message, map[string]any{
		"format":           current,
		"requires_restart": !strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(applied)),
	})
}
