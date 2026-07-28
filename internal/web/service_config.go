package web

import (
	"net/http"
	"os"
	"strings"

	"arupa/internal/conf"
	"arupa/internal/netx"
	"arupa/internal/service"
)

type serviceDirRequest struct {
	ServiceDir string `json:"service_dir"`
}

type serviceTempDirRequest struct {
	TempDir string `json:"temp_dir"`
}

type serviceConfigPatch struct {
	Restart   optionalField[string]   `json:"restart"`
	RunAsUser optionalField[string]   `json:"run_as_user"`
	Checksum  optionalField[string]   `json:"checksum"`
	Allow     optionalField[[]string] `json:"allow"`
}

type serviceConfigResponse struct {
	Name       string   `json:"name"`
	Configured bool     `json:"configured"`
	Restart    string   `json:"restart,omitempty"`
	RunAsUser  string   `json:"run_as_user,omitempty"`
	Checksum   string   `json:"checksum,omitempty"`
	Allow      []string `json:"allow"`
}

type serviceParamsPatch struct {
	Set    map[string]string `json:"set,omitempty"`
	Remove []string          `json:"remove,omitempty"`
}

func startServiceConfig(mux *http.ServeMux, sm *service.Manager) {
	mux.HandleFunc("GET /api/service/dir", management(conf.APICapabilityService, handleServiceDir))
	mux.HandleFunc("PATCH /api/service/dir", management(conf.APICapabilityService, func(w http.ResponseWriter, r *http.Request) {
		handlePatchServiceDir(w, r, sm)
	}))
	mux.HandleFunc("GET /api/service/temp-dir", management(conf.APICapabilityService, func(w http.ResponseWriter, _ *http.Request) {
		writeServiceTempDir(w, sm, "Service temporary directory fetched")
	}))
	mux.HandleFunc("PATCH /api/service/temp-dir", management(conf.APICapabilityService, func(w http.ResponseWriter, r *http.Request) {
		handlePatchServiceTempDir(w, r, sm)
	}))
	mux.HandleFunc("GET /api/service/config/{name}", management(conf.APICapabilityService, handleServiceEntryConfig))
	mux.HandleFunc("PATCH /api/service/config/{name}", management(conf.APICapabilityService, handlePatchServiceEntryConfig))
	mux.HandleFunc("DELETE /api/service/config/{name}", management(conf.APICapabilityService, handleDeleteServiceEntryConfig))
	mux.HandleFunc("GET /api/service/params/{name}", management(conf.APICapabilityService, handleServiceParams))
	mux.HandleFunc("PATCH /api/service/params/{name}", management(conf.APICapabilityService, handlePatchServiceParams))
}

func handleServiceDir(w http.ResponseWriter, _ *http.Request) {
	serviceDir, _ := conf.GetServicePaths()
	_ = netx.WriteSuccess(w, "Service directory fetched", map[string]string{
		"service_dir": serviceDir,
	})
}

func handlePatchServiceDir(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	var request serviceDirRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	request.ServiceDir = strings.TrimSpace(request.ServiceDir)
	if request.ServiceDir == "" {
		_ = netx.WriteBadRequest(w, "service_dir is required")
		return
	}
	if err := os.MkdirAll(request.ServiceDir, 0o755); err != nil {
		_ = netx.WriteBadRequest(w, "Invalid service_dir")
		return
	}
	if err := conf.Update(conf.Set(
		conf.JoinPath(string(conf.ConfigFieldServiceDir)), request.ServiceDir,
	)); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update service directory", err)
		return
	}
	if err := sm.Scan(); err != nil {
		_ = netx.WriteInternalServerError(w, "Service directory updated, but scan failed", err)
		return
	}
	serviceDir, _ := conf.GetServicePaths()
	_ = netx.WriteSuccess(w, "Service directory updated", map[string]any{
		"service_dir":              serviceDir,
		"discovered_service_count": len(sm.Entries()),
	})
}

func handlePatchServiceTempDir(w http.ResponseWriter, r *http.Request, sm *service.Manager) {
	var request serviceTempDirRequest
	if !decodeRequest(w, r, &request) {
		return
	}
	request.TempDir = strings.TrimSpace(request.TempDir)
	if request.TempDir == "" {
		_ = netx.WriteBadRequest(w, "temp_dir is required")
		return
	}
	if err := os.MkdirAll(request.TempDir, 0o755); err != nil {
		_ = netx.WriteBadRequest(w, "Invalid temp_dir")
		return
	}
	if err := conf.Update(conf.Set(
		conf.JoinPath(string(conf.ConfigFieldServiceTempDir)), request.TempDir,
	)); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update service temporary directory", err)
		return
	}
	writeServiceTempDir(w, sm, "Service temporary directory updated")
}

func writeServiceTempDir(w http.ResponseWriter, sm *service.Manager, message string) {
	_, tempDir := conf.GetServicePaths()
	_ = netx.WriteSuccess(w, message, map[string]any{
		"temp_dir":         tempDir,
		"requires_restart": sm.TempDirRequiresRestart(),
	})
}

func handleServiceEntryConfig(w http.ResponseWriter, r *http.Request) {
	name, ok := serviceNameFromPath(w, r)
	if !ok {
		return
	}
	configs, found := conf.GetServices(name)
	if !found[0] {
		_ = netx.WriteSuccess(w, "Service config fetched", serviceConfigResponse{Name: name})
		return
	}
	_ = netx.WriteSuccess(w, "Service config fetched", newServiceConfigResponse(name, configs[0], true))
}

func handlePatchServiceEntryConfig(w http.ResponseWriter, r *http.Request) {
	name, ok := serviceNameFromPath(w, r)
	if !ok {
		return
	}
	var request serviceConfigPatch
	if !decodeRequest(w, r, &request) {
		return
	}

	operations := make([]conf.Operation, 0, 4)
	appendStringPatch := func(field conf.ServiceField, value optionalField[string]) {
		if !value.Present {
			return
		}
		path := conf.JoinPath(string(conf.ConfigFieldServices), name, string(field))
		if value.Null || strings.TrimSpace(value.Value) == "" {
			operations = append(operations, conf.Remove(path))
		} else {
			operations = append(operations, conf.Set(path, strings.TrimSpace(value.Value)))
		}
	}
	appendStringPatch(conf.ServiceFieldRestart, request.Restart)
	appendStringPatch(conf.ServiceFieldRunAsUser, request.RunAsUser)
	appendStringPatch(conf.ServiceFieldChecksum, request.Checksum)
	if request.Allow.Present {
		path := conf.JoinPath(string(conf.ConfigFieldServices), name, string(conf.ServiceFieldAllow))
		if request.Allow.Null {
			operations = append(operations, conf.Remove(path))
		} else {
			operations = append(operations, conf.Set(path, append([]string(nil), request.Allow.Value...)))
		}
	}
	if len(operations) == 0 {
		_ = netx.WriteBadRequest(w, "At least one service config field is required")
		return
	}
	if err := conf.Update(operations...); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update service config", err)
		return
	}
	configs, found := conf.GetServices(name)
	_ = netx.WriteSuccess(w, "Service config updated", newServiceConfigResponse(name, configs[0], found[0]))
}

func handleDeleteServiceEntryConfig(w http.ResponseWriter, r *http.Request) {
	name, ok := serviceNameFromPath(w, r)
	if !ok {
		return
	}
	if err := conf.Update(conf.Remove(conf.JoinPath(string(conf.ConfigFieldServices), name))); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to delete service config", err)
		return
	}
	_ = netx.WriteSuccess(w, "Service config deleted", map[string]string{"name": name})
}

func newServiceConfigResponse(name string, config conf.Service, configured bool) serviceConfigResponse {
	var allow []string
	if config.Allow != nil {
		allow = append([]string{}, config.Allow...)
	}
	return serviceConfigResponse{
		Name: name, Configured: configured, Restart: config.Restart,
		RunAsUser: config.RunAsUser, Checksum: config.Checksum,
		Allow: allow,
	}
}

func handleServiceParams(w http.ResponseWriter, r *http.Request) {
	name, ok := serviceNameFromPath(w, r)
	if !ok {
		return
	}
	configs, _ := conf.GetServices(name)
	_ = netx.WriteSuccess(w, "Service params fetched", map[string]any{
		"name":   name,
		"params": configs[0].Params,
	})
}

func handlePatchServiceParams(w http.ResponseWriter, r *http.Request) {
	name, ok := serviceNameFromPath(w, r)
	if !ok {
		return
	}
	var request serviceParamsPatch
	if !decodeRequest(w, r, &request) {
		return
	}
	operations := make([]conf.Operation, 0, len(request.Remove)+len(request.Set))
	for _, key := range request.Remove {
		operations = append(operations, conf.Remove(conf.JoinPath(
			string(conf.ConfigFieldServices), name, string(conf.ServiceFieldParams), key,
		)))
	}
	for key, value := range request.Set {
		operations = append(operations, conf.Set(conf.JoinPath(
			string(conf.ConfigFieldServices), name, string(conf.ServiceFieldParams), key,
		), value))
	}
	if len(operations) == 0 {
		_ = netx.WriteBadRequest(w, "At least one params operation is required")
		return
	}
	if err := conf.Update(operations...); err != nil {
		_ = netx.WriteInternalServerError(w, "Failed to update service params", err)
		return
	}
	configs, _ := conf.GetServices(name)
	_ = netx.WriteSuccess(w, "Service params updated", map[string]any{
		"name":   name,
		"params": configs[0].Params,
	})
}
