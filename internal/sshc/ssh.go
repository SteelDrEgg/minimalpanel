package sshc

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
)

func loadConfig(host string) (username string, hostName string, port string, err error) {
	// Load user's SSH config file
	f, err := os.Open(os.ExpandEnv("$HOME/.ssh/config"))
	sshConfig, err := ssh_config.Decode(f)
	if err != nil {
		log.Fatalf("Failed to load SSH config: %s", err)
	}

	// Get the host configuration
	username, err = sshConfig.Get(host, "User")
	if err != nil {
		log.Fatalf("Failed to get ssh User from config: %s", err)
	}

	hostName, err = sshConfig.Get(host, "HostName")
	if err != nil {
		log.Fatalf("Failed to get ssh HostName from config: %s", err)
	}

	port, err = sshConfig.Get(host, "Port")
	if err != nil {
		log.Fatalf("Failed to get ssh Port from config: %s", err)
	}

	return username, hostName, port, nil
}

// loadCert loads a private key for SSH authentication
// keyPath: path to the private key file
// passphrase: optional passphrase for encrypted keys (can be nil or empty)
// Returns ssh.Signer and error
func loadCert(keyPath string, passphrase string) (ssh.Signer, error) {
	if keyPath == "" {
		// This probably won't work for www user
		keyPath = "$HOME/.ssh/id_rsa"
	}
	keyPath = os.ExpandEnv(keyPath)

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
	if len(passphrase) > 0 {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
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

func Client() {
	//user, hostName, port, err := loadConfig("claw1")
	user := "root"
	hostName := "47.251.7.109"
	//passwd := "f0E97Y+l[5IRhz"
	host := "claw1"
	key, _ := loadCert("", "bill1212")
	sshClientConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
			//ssh.Password(passwd),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	address := net.JoinHostPort(hostName, "22")
	client, err := ssh.Dial("tcp", address, sshClientConfig)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %s", host, err)
	}

	fmt.Printf("Connected to %s\n", host)
	defer client.Close()

	// 创建 SSH 会话
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("failed to create session: %s", err)
	}
	defer session.Close()

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
