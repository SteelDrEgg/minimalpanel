package sshc

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
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

// loadAuth creates SSH authentication methods based on provided credentials
// password: optional password for password authentication
// identities: optional slice of identity structs for public key authentication
// Returns a slice of ssh.AuthMethod that can be used for SSH authentication
func loadAuth(password string, identities []*identity) ([]ssh.AuthMethod, error) {
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

// connect creates SSH connection
// host: host information for connection
// auth: credential for connection
// Returns pointer to ssh connection
func connect(host *Host, auth []ssh.AuthMethod) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            host.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         host.Timeout,
	}
	addr := net.JoinHostPort(host.Hostname, host.Port)

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %s: %v\n", addr, err)
	}

	return client, err
}

func stdoutPrint(stdout io.Reader) {
	for {
		buffer := make([]byte, 1024)
		_, err := stdout.Read(buffer)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
		}
		fmt.Println(string(buffer))
	}
}

// TODO: not done yet, continue working on it
func Client() {
	key, _ := loadAuth("", []*identity{{keyPath: "$HOME/.ssh/id_rsa", passphrase: "1234"}})
	config, _ := loadConfig("claw1", "")
	client, err := connect(config, key)
	if err != nil {
		fmt.Println("Failed to connect to Claw1")
		os.Exit(1)
	}
	defer client.Close()

	session, err := client.NewSession()

	stdin, stdout, err := setupTerminal(session, 10, 20)
	session.Shell()

	go stdoutPrint(stdout)
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		if text == "wc" {
			session.WindowChange(10, 50)
		} else {
			stdin.Write([]byte(text))
		}
	}

	fmt.Println("session closed")
	session.Close()
	return
}
