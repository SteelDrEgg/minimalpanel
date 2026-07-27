package web

import (
	"arupa/internal/auth"
	"arupa/internal/conf"
	"arupa/internal/netx"
	"arupa/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

// serviceActionRequest is the common JSON payload for start/stop/restart.
type serviceActionRequest struct {
	Name string `json:"name"`
}

// serviceConfigRequest carries service directory settings from the management UI.
type serviceConfigRequest struct {
	ServiceDir     string `json:"service_dir"`
	ServiceTempDir string `json:"service_temp_dir"`
}

// serviceView is the catalog row returned to the management UI. It combines
// scanned package metadata with the current runtime status.
type serviceView struct {
	Name            string                `json:"name"`
	Version         string                `json:"version"`
	Type            string                `json:"type"`
	ContractVersion int                   `json:"contract_version"`
	Command         string                `json:"command"`
	PackagePath     string                `json:"package_path"`
	Config          serviceConfigView     `json:"config"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
	Status          service.ServiceStatus `json:"status"`
}

// serviceConfigView deliberately keeps Params separate from the normal list
// response. Params may be large or contain credentials.
type serviceConfigView struct {
	Restart   string            `json:"restart"`
	RunAsUser string            `json:"run_as_user,omitempty"`
	Checksum  string            `json:"checksum,omitempty"`
	Allow     []string          `json:"allow,omitempty"`
	Params    map[string]string `json:"params,omitempty"`
}

type serviceListInclude struct {
	metadata bool
	params   bool
}

// StartService registers service management endpoints used by frontend pages.
// All service endpoints are protected.
func StartService(mux *http.ServeMux, sm *service.Manager) {
	mux.HandleFunc("/api/services/discovered", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		handleServiceDiscovered(w, r, sm)
	}))
	mux.HandleFunc("/api/services/running", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		handleServiceRunning(w, r, sm)
	}))
	mux.HandleFunc("/api/services/start", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		handleServiceStart(w, r, sm)
	}))
	mux.HandleFunc("/api/services/stop", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		handleServiceStop(w, r, sm)
	}))
	mux.HandleFunc("/api/services/restart", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		handleServiceRestart(w, r, sm)
	}))
	mux.HandleFunc("/api/services/config", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		handleServiceConfig(w, r, sm)
	}))
}

// handleServiceDiscovered scans the service directory and returns the
// lightweight discovered-service projection.
func handleServiceDiscovered(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	if r.Method != http.MethodGet {
		netx.WriteMethodNotAllowed(w)
		return
	}

	include, err := parseServiceListInclude(r)
	if err != nil {
		netx.WriteBadRequest(w, err.Error())
		return
	}

	if err := sm.Scan(); err != nil {
		netx.WriteInternalServerError(w, "Failed to read service directory", err)
		return
	}
	entries := sm.Entries()
	services := make([]serviceView, 0, len(entries))
	for _, entry := range entries {
		services = append(services, newServiceView(entry, include))
	}

	netx.WriteSuccess(w, "Discovered services fetched", map[string]any{
		"services": services,
	})
}

// handleServiceRunning returns registered runtime instances without scanning
// the service directory. Records may include routes and transports.
func handleServiceRunning(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	if r.Method != http.MethodGet {
		netx.WriteMethodNotAllowed(w)
		return
	}

	running := sm.Registry().List()
	sort.Slice(running, func(i, j int) bool {
		return running[i].InstanceID < running[j].InstanceID
	})
	netx.WriteSuccess(w, "Running services fetched", map[string]any{
		"services": running,
	})
}

// parseServiceListInclude parses additive response fields from repeated and
// comma-separated include query parameters. The default response contains
// neither metadata nor params.
func parseServiceListInclude(r *http.Request) (serviceListInclude, error) {
	var include serviceListInclude
	for _, value := range r.URL.Query()["include"] {
		for _, field := range strings.Split(value, ",") {
			switch strings.ToLower(strings.TrimSpace(field)) {
			case "":
				continue
			case "metadata":
				include.metadata = true
			case "params":
				include.params = true
			default:
				return serviceListInclude{}, fmt.Errorf("unsupported include field %q", field)
			}
		}
	}
	return include, nil
}

func newServiceView(entry service.ServiceEntry, include serviceListInclude) serviceView {
	config := serviceConfigView{
		Restart:   entry.Config.Restart,
		RunAsUser: entry.Config.RunAsUser,
		Checksum:  entry.Config.Checksum,
		Allow:     entry.Config.Allow,
	}
	if include.params {
		config.Params = entry.Config.Params
	}

	view := serviceView{
		Name:            entry.Name,
		Version:         entry.Version,
		Type:            entry.Type,
		ContractVersion: entry.ContractVersion,
		Command:         entry.Command,
		PackagePath:     entry.PackagePath,
		Config:          config,
		Status:          entry.Status,
	}
	if include.metadata {
		view.Metadata = entry.Metadata
	}
	return view
}

func handleServiceStart(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	if r.Method != http.MethodPost {
		netx.WriteMethodNotAllowed(w)
		return
	}

	name, ok := readServiceActionName(w, r)
	if !ok {
		return
	}

	if err := sm.Start(name); err != nil {
		netx.WriteBadRequest(w, fmt.Sprintf("Failed to start service %q: %v", name, err))
		return
	}

	netx.WriteSuccess(w, "Service started", map[string]any{
		"name": name,
	})
}

func handleServiceStop(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	if r.Method != http.MethodPost {
		netx.WriteMethodNotAllowed(w)
		return
	}

	name, ok := readServiceActionName(w, r)
	if !ok {
		return
	}

	if err := sm.Stop(name); err != nil {
		netx.WriteBadRequest(w, fmt.Sprintf("Failed to stop service %q: %v", name, err))
		return
	}

	netx.WriteSuccess(w, "Service stopped", map[string]any{
		"name": name,
	})
}

func handleServiceRestart(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	if r.Method != http.MethodPost {
		netx.WriteMethodNotAllowed(w)
		return
	}

	name, ok := readServiceActionName(w, r)
	if !ok {
		return
	}

	if err := sm.Restart(name); err != nil {
		netx.WriteBadRequest(w, fmt.Sprintf("Failed to restart service %q: %v", name, err))
		return
	}

	netx.WriteSuccess(w, "Service restarted", map[string]any{
		"name": name,
	})
}

func readServiceActionName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req serviceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		netx.WriteBadRequest(w, "Invalid request body")
		return "", false
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		netx.WriteBadRequest(w, "Service name is required")
		return "", false
	}
	return name, true
}

func handleServiceConfig(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	switch r.Method {
	case http.MethodGet:
		serviceDir, serviceTempDir := conf.GetServicePaths()
		netx.WriteSuccess(w, "Service config fetched", map[string]any{
			"service_dir":      serviceDir,
			"service_temp_dir": serviceTempDir,
		})
	case http.MethodPut:
		var req serviceConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			netx.WriteBadRequest(w, "Invalid request body")
			return
		}

		req.ServiceDir = strings.TrimSpace(req.ServiceDir)
		req.ServiceTempDir = strings.TrimSpace(req.ServiceTempDir)
		if req.ServiceDir == "" || req.ServiceTempDir == "" {
			netx.WriteBadRequest(w, "service_dir and service_temp_dir are required")
			return
		}

		// Validate/create directories first, then persist config.
		if err := os.MkdirAll(req.ServiceDir, 0o755); err != nil {
			netx.WriteBadRequest(w, fmt.Sprintf("Invalid service_dir: %v", err))
			return
		}
		if err := os.MkdirAll(req.ServiceTempDir, 0o755); err != nil {
			netx.WriteBadRequest(w, fmt.Sprintf("Invalid service_temp_dir: %v", err))
			return
		}

		_, oldServiceTempDir := conf.GetServicePaths()
		if err := conf.Update(
			conf.Set(conf.JoinPath(string(conf.ConfigFieldServiceDir)), req.ServiceDir),
			conf.Set(conf.JoinPath(string(conf.ConfigFieldServiceTempDir)), req.ServiceTempDir),
		); err != nil {
			netx.WriteInternalServerError(w, "Failed to persist service config", err)
			return
		}

		if err := sm.Scan(); err != nil {
			netx.WriteInternalServerError(w, "Service config saved, but scan failed", err)
			return
		}

		newServiceDir, newServiceTempDir := conf.GetServicePaths()
		tempDirChanged := oldServiceTempDir != newServiceTempDir
		netx.WriteSuccess(w, "Service config updated", map[string]any{
			"service_dir":                     newServiceDir,
			"service_temp_dir":                newServiceTempDir,
			"temp_dir_requires_restart":       tempDirChanged,
			"temp_dir_restart_reason":         "running services keep their current extraction directory",
			"discovered_service_count":        len(sm.Entries()),
			"scan_path_effective_immediately": true,
		})
	default:
		netx.WriteMethodNotAllowed(w)
	}
}
