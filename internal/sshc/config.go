package sshc

import (
	"fmt"
	"github.com/kevinburke/ssh_config"
	"github.com/spf13/cast"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadConfig loads SSH configuration for a specific host from SSH config file
// hostAlias: the SSH host alias to look up
// configPath: optional path to SSH config file (empty string uses default ~/.ssh/config)
// Returns a Host struct with all relevant configuration options
func LoadConfig(hostAlias string, configPath string) (*Host, error) {
	if configPath == "" {
		configPath = "$HOME/.ssh/config"
	}
	configPath = os.ExpandEnv(configPath)

	// Read config
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH config file %s: %w", configPath, err)
	}
	defer f.Close()
	// Parse config
	sshConfig, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Helper function to get config value with default
	getValue := func(key, defaultValue string) string {
		value, _ := sshConfig.Get(hostAlias, key)
		if value == "" {
			return defaultValue
		}
		return value
	}

	host := &Host{
		Host:         hostAlias,
		User:         getValue("User", os.Getenv("USER")),
		Hostname:     getValue("HostName", hostAlias),
		Port:         getValue("Port", "22"),
		IdentityFile: getValue("IdentityFile", "$HOME/.ssh/id_rsa"),
		Timeout:      cast.ToDuration(getValue("ConnectTimeout", "10")) * time.Second,
	}

	host.IdentityFile = os.ExpandEnv(host.IdentityFile)

	return host, nil
}

// saveConfig saves a Host configuration to the SSH config file
// host: the Host struct to save
// configPath: optional path to SSH config file (empty string uses default ~/.ssh/config)
// Returns error if any operation fails
func saveConfig(host *Host, configPath string) error {
	if configPath == "" {
		configPath = "$HOME/.ssh/config"
	}
	configPath = os.ExpandEnv(configPath)

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH config directory %s: %w", configDir, err)
	}

	// Load existing config if file exists
	var cfg *ssh_config.Config
	f, err := os.Open(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to open SSH config file %s: %w", configPath, err)
		}
		// Create new config if file doesn't exist
		cfg = &ssh_config.Config{}
	} else {
		defer f.Close()
		cfg, err = ssh_config.Decode(f)
		if err != nil {
			return fmt.Errorf("failed to parse SSH config: %w", err)
		}
	}

	// Create or update host configuration
	pattern, err := ssh_config.NewPattern(host.Host)
	if err != nil {
		return fmt.Errorf("failed to create pattern for host %s: %w", host.Host, err)
	}

	hostConfig := &ssh_config.Host{
		Patterns: []*ssh_config.Pattern{pattern},
		Nodes:    []ssh_config.Node{},
	}

	// Helper function to add non-empty fields
	addField := func(key, value string) {
		if value != "" {
			hostConfig.Nodes = append(hostConfig.Nodes, &ssh_config.KV{
				Key:   key,
				Value: value,
			})
		}
	}

	// Add non-empty fields to the host configuration
	addField("User", host.User)
	if host.Hostname != host.Host {
		addField("HostName", host.Hostname)
	}
	addField("Port", host.Port)
	addField("IdentityFile", host.IdentityFile)
	addField("ConnectTimeout", cast.ToString(host.Timeout))

	// TODO: Allow multiple fields with same key
	appendOnlyFields := map[string]bool{
		//"IdentityFile": true,
	}

	// Find existing host configuration
	var targetHost *ssh_config.Host
	for _, h := range cfg.Hosts {
		for _, pattern := range h.Patterns {
			if pattern.String() == host.Host {
				targetHost = h
				break
			}
		}
		if targetHost != nil {
			break
		}
	}

	// If host not found, add new host config
	if targetHost == nil {
		cfg.Hosts = append(cfg.Hosts, hostConfig)
	} else {
		// Find all pre-exist fields, skip whitelist fields from index
		kvIndex := make(map[string]int, len(targetHost.Nodes))
		for i, node := range targetHost.Nodes {
			if kv, ok := node.(*ssh_config.KV); ok && !appendOnlyFields[kv.Key] {
				kvIndex[kv.Key] = i
			}
		}

		for _, newNode := range hostConfig.Nodes {
			// Non-KV
			if _, ok := newNode.(*ssh_config.KV); !ok {
				targetHost.Nodes = append(targetHost.Nodes, newNode)
				continue
			}

			// KV node
			newKV := newNode.(*ssh_config.KV)
			if i, ok := kvIndex[newKV.Key]; ok {
				targetHost.Nodes[i] = newKV
			} else {
				// New field or whitelist field: append
				if !appendOnlyFields[newKV.Key] {
					kvIndex[newKV.Key] = len(targetHost.Nodes)
				}
				targetHost.Nodes = append(targetHost.Nodes, newKV)
			}
		}
	}

	// Write the config back to file
	configContent := cfg.String()

	// Remove any empty lines at the beginning of the file
	configContent = strings.TrimLeft(configContent, "\n")

	// Write to file
	return ioutil.WriteFile(configPath, []byte(configContent), 0600)
}
