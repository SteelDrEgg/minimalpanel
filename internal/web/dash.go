package web

import (
	"fmt"
	"minimalpanel/internal/auth"
	"minimalpanel/internal/netx"
	"minimalpanel/internal/system"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zishang520/socket.io/servers/socket/v3"
)

// DashboardSession represents an active dashboard session
type DashboardSession struct {
	Socket      *socket.Socket
	RefreshRate time.Duration
	ticker      *time.Ticker
	cancelFunc  func() // Function to cancel current goroutine
	mutex       sync.Mutex
	active      bool
}

// DashboardSessionManager manages multiple dashboard sessions
type DashboardSessionManager struct {
	sessions map[string]*DashboardSession
	mutex    sync.RWMutex
}

var dashboardManager = &DashboardSessionManager{
	sessions: make(map[string]*DashboardSession),
}

// SystemMetrics represents the dynamic system metrics data (CPU, Memory, Disk)
type SystemMetrics struct {
	CPU    CPUMetric    `json:"cpu"`
	Memory MemoryMetric `json:"memory"`
	Disk   DiskMetric   `json:"disk"`
}

// SystemInfo represents static system information
type SystemBasicInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	IP       string `json:"ip"`
	Username string `json:"username"`
}

type CPUMetric struct {
	Usage float64 `json:"usage"`
	Model string  `json:"model"`
	Cores int32   `json:"cores"`
}

type MemoryMetric struct {
	Used        string  `json:"used"`
	Total       string  `json:"total"`
	UsedPercent float64 `json:"used_percent"`
	Swap        string  `json:"swap"`
}

type DiskMetric struct {
	Used        string  `json:"used"`
	Total       string  `json:"total"`
	UsedPercent float64 `json:"used_percent"`
}

// SetupDashboardService sets up the dashboard socket.io namespace on the global server
func SetupDashboardService() {
	server := netx.GetGlobalServer()
	dashNamespace := server.GetNamespace("/dashboard")

	// Handle dashboard connection requests
	dashNamespace.AddEvent("connect_dashboard", handleDashboardConnect)

	// Handle refresh rate changes
	dashNamespace.AddEvent("set_refresh_rate", handleSetRefreshRate)

	// Handle manual refresh requests
	dashNamespace.AddEvent("refresh_data", handleRefreshData)

	// Handle disconnect (standard Socket.IO event)
	dashNamespace.AddEvent("disconnect", handleDashboardDisconnect)

	dashNamespace.RegisterEvents()

	// Auth - commented out for development/testing
	dashNamespace.AddMiddleware(auth.RequireAuthSocketIO)
}

// handleDashboardConnect handles dashboard connection requests
func handleDashboardConnect(client *socket.Socket, data ...any) {
	fmt.Printf("Dashboard client connected: %s\n", client.Id())

	// Get username from authentication
	username := getUsernameFromSocket(client)

	// Create dashboard session with default 10s refresh rate
	session := &DashboardSession{
		Socket:      client,
		RefreshRate: 10 * time.Second,
		active:      true,
	}

	// Store session
	dashboardManager.mutex.Lock()
	dashboardManager.sessions[string(client.Id())] = session
	dashboardManager.mutex.Unlock()

	// Send basic system info (one time)
	sendBasicSystemInfo(client, username)

	// Send initial metrics
	sendSystemMetrics(client)

	// Start periodic updates
	startPeriodicUpdates(session)

	// Emit connection success
	client.Emit("dashboard_connected", map[string]interface{}{
		"refresh_rate": "10s",
		"status":       "connected",
	})
}

// handleSetRefreshRate handles refresh rate changes
func handleSetRefreshRate(client *socket.Socket, data ...any) {
	if len(data) == 0 {
		client.Emit("dashboard_error", "No refresh rate data provided")
		return
	}

	var rateData map[string]interface{}
	var ok bool

	// Handle potential nested array format (similar to SSH implementation)
	if rateData, ok = data[0].(map[string]interface{}); !ok {
		if dataArray, isArray := data[0].([]interface{}); isArray && len(dataArray) > 0 {
			rateData, ok = dataArray[0].(map[string]interface{})
		}
	}

	if !ok {
		client.Emit("dashboard_error", "Invalid refresh rate data format")
		return
	}

	rateStr, _ := rateData["rate"].(string)
	if rateStr == "" {
		client.Emit("dashboard_error", "Refresh rate is required")
		return
	}

	dashboardManager.mutex.RLock()
	session, exists := dashboardManager.sessions[string(client.Id())]
	dashboardManager.mutex.RUnlock()

	if !exists || !session.active {
		client.Emit("dashboard_error", "No active dashboard session")
		return
	}

	// Parse refresh rate
	var newRate time.Duration

	if rateStr == "OFF" {
		newRate = 0
	} else if len(rateStr) > 1 && rateStr[len(rateStr)-1] == 's' {
		seconds, parseErr := strconv.Atoi(rateStr[:len(rateStr)-1])
		if parseErr != nil {
			client.Emit("dashboard_error", "Invalid refresh rate format")
			return
		}
		newRate = time.Duration(seconds) * time.Second
	} else {
		client.Emit("dashboard_error", "Invalid refresh rate format")
		return
	}

	// Update session refresh rate
	session.mutex.Lock()

	// Stop current ticker and cancel goroutine
	if session.ticker != nil {
		session.ticker.Stop()
		session.ticker = nil
	}
	if session.cancelFunc != nil {
		session.cancelFunc()
		session.cancelFunc = nil
	}

	session.RefreshRate = newRate
	session.mutex.Unlock()

	// Start new periodic updates if rate is not OFF
	if newRate > 0 {
		startPeriodicUpdates(session)
	}

	client.Emit("refresh_rate_updated", map[string]interface{}{
		"rate": rateStr,
	})
}

// handleRefreshData handles manual refresh requests
func handleRefreshData(client *socket.Socket, data ...any) {
	sendSystemMetrics(client)
}

// handleDashboardDisconnect handles dashboard disconnection
func handleDashboardDisconnect(client *socket.Socket, data ...any) {
	cleanupDashboardSession(string(client.Id()))
}

// startPeriodicUpdates starts sending periodic system metrics updates
func startPeriodicUpdates(session *DashboardSession) {
	if session.RefreshRate <= 0 {
		return
	}

	session.mutex.Lock()
	// Stop existing ticker if any
	if session.ticker != nil {
		session.ticker.Stop()
	}
	if session.cancelFunc != nil {
		session.cancelFunc()
	}

	// Create new ticker and cancel function
	ticker := time.NewTicker(session.RefreshRate)
	session.ticker = ticker

	// Create a done channel for this specific goroutine
	done := make(chan struct{})
	session.cancelFunc = func() {
		close(done)
	}
	session.mutex.Unlock()

	go func() {
		defer func() {
			ticker.Stop()
			session.mutex.Lock()
			if session.ticker == ticker {
				session.ticker = nil
			}
			if session.cancelFunc != nil {
				// Clear cancel function if it's for this goroutine
				session.cancelFunc = nil
			}
			session.mutex.Unlock()
		}()

		for {
			select {
			case <-ticker.C:
				session.mutex.Lock()
				active := session.active
				socket := session.Socket
				session.mutex.Unlock()

				if !active {
					return
				}
				if socket != nil {
					sendSystemMetrics(socket)
				}

			case <-done:
				return
			}
		}
	}()
}

// getUsernameFromSocket extracts username from socket authentication
func getUsernameFromSocket(client *socket.Socket) string {
	// Try to get username from cookie (if auth is enabled)
	cookieHeader := client.Handshake().Headers["Cookie"]
	if cookieHeader != nil {
		if cookieSlice, ok := cookieHeader.([]string); ok && len(cookieSlice) > 0 {
			cookies := cookieSlice[0]
			parts := strings.Split(cookies, ";")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, auth.CookieName+"=") {
					token := strings.TrimPrefix(p, auth.CookieName+"=")
					if username, valid := auth.ValidateSession(token); valid {
						return username
					}
				}
			}
		}
	}
	return "Administrator" // Default fallback
}

// sendBasicSystemInfo sends static system information (one time)
func sendBasicSystemInfo(client *socket.Socket, username string) {
	info, err := system.GetSystemInfo()
	if err != nil {
		client.Emit("dashboard_error", fmt.Sprintf("Failed to get system info: %v", err))
		return
	}

	hostname, err := system.GetHostInfo()
	if err != nil {
		hostname = "unknown"
	}

	basicInfo := &SystemBasicInfo{
		Hostname: hostname,
		OS:       info.OS,
		IP:       "127.0.0.1", // Placeholder - you can implement actual IP detection
		Username: username,
	}

	client.Emit("basic_system_info", basicInfo)
}

// sendSystemMetrics sends current system metrics to the client
func sendSystemMetrics(client *socket.Socket) {
	metrics, err := collectSystemMetrics()
	if err != nil {
		client.Emit("dashboard_error", fmt.Sprintf("Failed to collect system metrics: %v", err))
		return
	}

	client.Emit("system_metrics", metrics)
}

// collectSystemMetrics collects current system metrics using our internal/system package
func collectSystemMetrics() (*SystemMetrics, error) {
	// Get system usage
	usage, err := system.GetSystemUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get system usage: %w", err)
	}

	metrics := &SystemMetrics{
		CPU: CPUMetric{
			Usage: usage.CPU.Percent,
			Model: usage.CPU.Model,
			Cores: usage.CPU.Cores,
		},
		Memory: MemoryMetric{
			Used:        system.ProperUnit(usage.Memory.Used),
			Total:       system.ProperUnit(usage.Memory.Total),
			UsedPercent: usage.Memory.UsedPercent,
			Swap:        system.ProperUnit(usage.Memory.Swap),
		},
		Disk: DiskMetric{
			Used:        system.ProperUnit(usage.Disk.Used),
			Total:       system.ProperUnit(usage.Disk.Total),
			UsedPercent: usage.Disk.UsedPercent,
		},
	}

	return metrics, nil
}

// cleanupDashboardSession cleans up a dashboard session
func cleanupDashboardSession(clientId string) {
	dashboardManager.mutex.Lock()
	defer dashboardManager.mutex.Unlock()

	session, exists := dashboardManager.sessions[clientId]
	if !exists {
		return
	}

	session.mutex.Lock()
	session.active = false

	// Stop ticker and cancel goroutine
	if session.ticker != nil {
		session.ticker.Stop()
		session.ticker = nil
	}
	if session.cancelFunc != nil {
		session.cancelFunc()
		session.cancelFunc = nil
	}

	session.mutex.Unlock()

	delete(dashboardManager.sessions, clientId)
	fmt.Printf("Dashboard client disconnected: %s\n", clientId)
}

// StartDashboard starts the dashboard service (deprecated - use SetupDashboardService instead)
func StartDashboard() {
	SetupDashboardService()
}

// GetActiveSessionsCount returns the number of active dashboard sessions
func GetActiveSessionsCount() int {
	dashboardManager.mutex.RLock()
	defer dashboardManager.mutex.RUnlock()
	return len(dashboardManager.sessions)
}
