package web

import (
	"bufio"
	"fmt"
	"io"
	"minimalpanel/internal/auth"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zishang520/socket.io/servers/socket/v3"
	"golang.org/x/crypto/ssh"
	"minimalpanel/internal/netx"
	"minimalpanel/internal/sshc"
)

// SSHSession represents an active SSH session with its connections
type SSHSession struct {
	Client  *ssh.Client
	Session *ssh.Session
	Stdin   io.WriteCloser
	Stdout  io.Reader
	Socket  *socket.Socket
	mutex   sync.Mutex
	active  bool
}

// SSHSessionManager manages multiple SSH sessions
type SSHSessionManager struct {
	sessions map[string]*SSHSession
	mutex    sync.RWMutex
}

var sessionManager = &SSHSessionManager{
	sessions: make(map[string]*SSHSession),
}

// CreateSSHServer sets up the SSH socket.io server
func CreateSSHServer() *netx.Socket {
	server := new(netx.Socket)
	server.Initialize()
	server.AddNamespace("/ssh")

	sshNamespace := server.GetNamespace("/ssh")

	// Handle SSH connection requests
	sshNamespace.AddEvent("connect_ssh", handleSSHConnect)

	// Handle terminal input
	sshNamespace.AddEvent("terminal_input", handleTerminalInput)

	// Handle window resize
	sshNamespace.AddEvent("resize", handleWindowResize)

	// Handle disconnect (standard Socket.IO event)
	sshNamespace.AddEvent("disconnect", handleSSHDisconnect)

	sshNamespace.RegisterEvents()

	// Auth
	sshNamespace.AddMiddleware(func(client *socket.Socket, next func(*socket.ExtendedError)) {
		cookies := client.Handshake().Headers["Cookie"].([]string)[0]
		cookie := func() string {
			parts := strings.Split(cookies, ";")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, auth.CookieName+"=") {
					return strings.TrimPrefix(p, auth.CookieName+"=")
				}
			}
			return ""
		}()
		if _, ok := auth.ValidateSession(cookie); ok {
			next(nil)
		} else {
			next(socket.NewExtendedError("Unauthorized", ""))
		}
	})

	return server
}

// handleSSHConnect handles SSH connection requests
func handleSSHConnect(client *socket.Socket, data ...any) {

	if len(data) == 0 {
		client.Emit("ssh_error", "No connection data provided")
		return
	}

	// Handle the Socket.IO data format - it might be nested in an array
	var connData map[string]interface{}
	var ok bool

	// First check if data[0] is already the map we want
	if connData, ok = data[0].(map[string]interface{}); ok {
		// Direct map - this is what we expect normally
	} else if dataArray, isArray := data[0].([]interface{}); isArray && len(dataArray) > 0 {
		// Socket.IO sometimes wraps data in an additional array layer
		if connData, ok = dataArray[0].(map[string]interface{}); !ok {
			client.Emit("ssh_error", fmt.Sprintf("Invalid nested connection data format. Received type: %T", dataArray[0]))
			return
		}
	} else {
		client.Emit("ssh_error", fmt.Sprintf("Invalid connection data format. Received type: %T", data[0]))
		return
	}

	// Extract connection parameters
	host, _ := connData["host"].(string)
	port, _ := connData["port"].(string)
	username, _ := connData["username"].(string)
	password, _ := connData["password"].(string)
	privateKey, _ := connData["privateKey"].(string)
	passphrase, _ := connData["passphrase"].(string)

	if host == "" || username == "" {
		client.Emit("ssh_error", "Host and username are required")
		return
	}

	if port == "" {
		port = "22"
	}

	// Try to load SSH config for the host alias first
	var hostConfig *sshc.Host

	// First try to load from SSH config if it looks like a host alias
	if !strings.Contains(host, ".") && host != "localhost" {
		if configHost, configErr := sshc.LoadConfig(host, ""); configErr == nil {
			hostConfig = configHost
			// Override with provided values if they're different from config
			if username != configHost.User && username != "" {
				hostConfig.User = username
			}
			if port != "22" && port != configHost.Port {
				hostConfig.Port = port
			}
		}
	}

	// If no SSH config found or failed to load, create manual configuration
	if hostConfig == nil {
		hostConfig = &sshc.Host{
			User:     username,
			Host:     host,
			Hostname: host,
			Port:     port,
			Timeout:  30 * time.Second,
		}
	}

	// Prepare authentication methods
	var authMethods []ssh.AuthMethod
	var err error

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	// Handle private key authentication
	if privateKey != "" {
		var identities []*sshc.Identity
		identity := &sshc.Identity{
			KeyPath:    privateKey,
			Passphrase: passphrase,
		}
		identities = append(identities, identity)

		keyAuthMethods, err := sshc.LoadAuth("", identities)
		if err != nil {
			client.Emit("ssh_error", fmt.Sprintf("Failed to load private key authentication: %v", err))
			return
		}
		authMethods = append(authMethods, keyAuthMethods...)
	} else if hostConfig.IdentityFile != "" {
		// Try to use identity file from SSH config
		var identities []*sshc.Identity
		identity := &sshc.Identity{
			KeyPath:    hostConfig.IdentityFile,
			Passphrase: passphrase, // Use provided passphrase if any
		}
		identities = append(identities, identity)

		keyAuthMethods, err := sshc.LoadAuth("", identities)
		if err == nil {
			authMethods = append(authMethods, keyAuthMethods...)
		}
	}

	// If no authentication methods loaded yet, try default identity files
	if len(authMethods) == 0 && password == "" {
		defaultKeys := []string{
			"$HOME/.ssh/id_rsa",
			"$HOME/.ssh/id_ed25519",
			"$HOME/.ssh/id_ecdsa",
		}

		for _, keyPath := range defaultKeys {
			identity := &sshc.Identity{
				KeyPath:    keyPath,
				Passphrase: passphrase, // Use provided passphrase if any
			}

			keyAuthMethods, err := sshc.LoadAuth("", []*sshc.Identity{identity})
			if err == nil {
				authMethods = append(authMethods, keyAuthMethods...)
				break // Use the first working key
			}
		}
	}

	if len(authMethods) == 0 {
		client.Emit("ssh_error", "No valid authentication method provided. Please provide either a password or a valid private key.")
		return
	}

	// Connect to SSH server
	sshClient, err := sshc.Connect(hostConfig, authMethods)
	if err != nil {
		client.Emit("ssh_error", fmt.Sprintf("SSH connection failed: %v", err))
		return
	}

	// Create SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()
		client.Emit("ssh_error", fmt.Sprintf("Failed to create SSH session: %v", err))
		return
	}

	// Setup terminal
	stdin, stdout, err := sshc.SetupTerminal(session, 24, 80)
	if err != nil {
		session.Close()
		sshClient.Close()
		client.Emit("ssh_error", fmt.Sprintf("Failed to setup terminal: %v", err))
		return
	}

	// Start shell
	err = session.Shell()
	if err != nil {
		session.Close()
		sshClient.Close()
		client.Emit("ssh_error", fmt.Sprintf("Failed to start shell: %v", err))
		return
	}

	// Create SSH session object
	sshSession := &SSHSession{
		Client:  sshClient,
		Session: session,
		Stdin:   stdin,
		Stdout:  stdout,
		Socket:  client,
		active:  true,
	}

	// Store session
	sessionManager.mutex.Lock()
	sessionManager.sessions[string(client.Id())] = sshSession
	sessionManager.mutex.Unlock()

	// Start reading from stdout
	go func() {

		reader := bufio.NewReader(stdout)
		buffer := make([]byte, 1024)

		for sshSession.active {
			n, err := reader.Read(buffer)
			if err != nil {
				break
			}

			if n > 0 {
				data := string(buffer[:n])
				client.Emit("terminal_output", data)
			}
		}
	}()

	// Emit connection success
	client.Emit("ssh_connected", map[string]interface{}{
		"host": host,
		"port": port,
		"user": username,
	})

}

// handleTerminalInput handles input from the terminal
func handleTerminalInput(client *socket.Socket, data ...any) {
	if len(data) == 0 {
		return
	}

	var input string
	var ok bool

	// Handle potential nested array format
	if input, ok = data[0].(string); !ok {
		if dataArray, isArray := data[0].([]interface{}); isArray && len(dataArray) > 0 {
			input, ok = dataArray[0].(string)
		}
	}

	if !ok {
		return
	}

	sessionManager.mutex.RLock()
	session, exists := sessionManager.sessions[string(client.Id())]
	sessionManager.mutex.RUnlock()

	if !exists || !session.active {
		client.Emit("ssh_error", "No active SSH session")
		return
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.Stdin != nil {
		_, err := session.Stdin.Write([]byte(input))
		if err != nil {
			client.Emit("ssh_error", "Failed to send input")
		}
	}
}

// handleWindowResize handles terminal window resize
func handleWindowResize(client *socket.Socket, data ...any) {
	if len(data) == 0 {
		return
	}

	var resizeData map[string]interface{}
	var ok bool

	// Handle potential nested array format
	if resizeData, ok = data[0].(map[string]interface{}); !ok {
		if dataArray, isArray := data[0].([]interface{}); isArray && len(dataArray) > 0 {
			resizeData, ok = dataArray[0].(map[string]interface{})
		}
	}

	if !ok {
		return
	}

	cols, _ := resizeData["cols"].(float64)
	rows, _ := resizeData["rows"].(float64)

	sessionManager.mutex.RLock()
	session, exists := sessionManager.sessions[string(client.Id())]
	sessionManager.mutex.RUnlock()

	if !exists || !session.active {
		return
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.Session != nil {
		session.Session.WindowChange(int(rows), int(cols))
	}
}

// handleSSHDisconnect handles SSH disconnection
func handleSSHDisconnect(client *socket.Socket, data ...any) {
	cleanupSession(string(client.Id()))
}

// cleanupSession cleans up an SSH session
func cleanupSession(clientId string) {
	sessionManager.mutex.Lock()
	defer sessionManager.mutex.Unlock()

	session, exists := sessionManager.sessions[clientId]
	if !exists {
		return
	}

	session.mutex.Lock()
	session.active = false
	session.mutex.Unlock()

	// Close connections
	if session.Stdin != nil {
		session.Stdin.Close()
	}
	if session.Session != nil {
		session.Session.Close()
	}
	if session.Client != nil {
		session.Client.Close()
	}

	delete(sessionManager.sessions, clientId)
}

// StartSSH starts the SSH service
func StartSSH() {
	server := CreateSSHServer()
	http.Handle("/socket.io/", server.Handler())
}
