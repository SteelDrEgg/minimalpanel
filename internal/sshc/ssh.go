package sshc

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

type Host struct {
	User         string
	Host         string
	Port         string
	Hostname     string
	IdentityFile string
	Timeout      time.Duration
}

type identity struct {
	keyPath    string
	passphrase string
}

// String implements fmt.Stringer interface for pretty printing
func (h *Host) String() string {
	return fmt.Sprintf("Host{User: %s, Host: %s, Hostname: %s, Port: %s, IdentityFile: %s, Timeout: %s}",
		h.User, h.Host, h.Hostname, h.Port, h.IdentityFile, h.Timeout)
}

// loadKey loads a private key for SSH authentication
// keyPath: path to the private key file
// passphrase: optional passphrase for encrypted keys (can be nil or empty)
// Returns ssh.Signer and error
func loadKey(key *identity) (ssh.Signer, error) {
	if key.keyPath == "" {
		// This probably won't work for www user
		key.keyPath = "$HOME/.ssh/id_rsa"
	}
	keyPath := os.ExpandEnv(key.keyPath)

	// Check if exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("private key file does not exist: %s", keyPath)
	}

	// Read the private key file
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from %s: %w", keyPath, err)
	}

	// Parse key
	var signer ssh.Signer
	if len(key.passphrase) > 0 {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(key.passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key with passphrase: %w", err)
		}
	} else {
		signer, err = ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key (key may be encrypted and require a passphrase): %w", err)
		}
	}
	return signer, nil
}

// login creates SSH authentication methods based on provided credentials
// password: optional password for password authentication
// identities: optional slice of identity structs for public key authentication
// Returns a slice of ssh.AuthMethod that can be used for SSH authentication
func login(password string, identities []*identity) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// Add password authentication if password is provided
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	// Add public key authentication for each identity
	for _, id := range identities {
		if id == nil {
			continue
		}

		signer, err := loadKey(id)
		if err != nil {
			// Log the error but continue with other authentication methods
			log.Printf("Failed to load key from %s: %v", id.keyPath, err)
			continue
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Return error if no authentication methods were successfully created
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no valid authentication methods available")
	}

	return authMethods, nil
}

func Client() {
	key, _ := login("", []*identity{&identity{keyPath: "$HOME/.ssh/id_rsa", passphrase: "1234"}})
	config, _ := loadConfig("claw1", "")
	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            key,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         config.Timeout,
	}

	addr := net.JoinHostPort(config.Hostname, config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
	}
	defer client.Close()

	session, err := client.NewSession()

	// 在远程机器上运行命令
	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to get stdout: %s", err)
	}
	if err := session.Start("ls"); err != nil {
		log.Fatalf("failed to start command: %s", err)
	}

	output, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Fatalf("无法读取 stdout：％s", err)
	}
	if err := session.Wait(); err != nil {
		log.Fatalf("命令失败：％s", err)
	}
	fmt.Println(string(output))
}
