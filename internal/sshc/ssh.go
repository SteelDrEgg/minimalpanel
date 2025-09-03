package sshc

import (
	"fmt"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
)

type Host struct {
	User         string
	Host         string
	Port         string
	Hostname     string
	IdentityFile string
	Timeout      string
}

// String implements fmt.Stringer interface for pretty printing
func (h *Host) String() string {
	return fmt.Sprintf("Host{User: %s, Host: %s, Hostname: %s, Port: %s, IdentityFile: %s, Timeout: %s}",
		h.User, h.Host, h.Hostname, h.Port, h.IdentityFile, h.Timeout)
}

// loadConfig loads SSH configuration for a specific host from SSH config file
// hostAlias: the SSH host alias to look up
// configPath: optional path to SSH config file (empty string uses default ~/.ssh/config)
// Returns a Host struct with all relevant configuration options
func loadConfig(hostAlias string, configPath string) (*Host, error) {
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
		Timeout:      getValue("ConnectTimeout", "10"),
	}

	host.IdentityFile = os.ExpandEnv(host.IdentityFile)

	return host, nil
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

func login() {

}

func Client() {
	host, err := loadConfig("claw1", "")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// 打印 Host 信息
	fmt.Println("Loaded SSH Host configuration:")
	fmt.Println(host)
	////host, err := loadConfig("claw1", "")  // 使用默认配置文件
	////host, err := loadConfig("claw1", "/path/to/custom/ssh_config")  // 使用自定义配置文件
	//user := "root"
	//hostName := "47.251.7.109"
	////passwd := "f0E97Y+l[5IRhz"
	//host := "claw1"
	//key, _ := loadCert("", "bill1212")
	//sshClientConfig := &ssh.ClientConfig{
	//	User: user,
	//	Auth: []ssh.AuthMethod{
	//		ssh.PublicKeys(key),
	//		//ssh.Password(passwd),
	//	},
	//	HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	//	Timeout:         5 * time.Second,
	//}
	//address := net.JoinHostPort(hostName, "22")
	//client, err := ssh.Dial("tcp", address, sshClientConfig)
	//if err != nil {
	//	log.Fatalf("Failed to connect to %s: %s", host, err)
	//}
	//
	//fmt.Printf("Connected to %s\n", host)
	//defer client.Close()
	//
	//// 创建 SSH 会话
	//session, err := client.NewSession()
	//if err != nil {
	//	log.Fatalf("failed to create session: %s", err)
	//}
	//defer session.Close()
	//
	//// 在远程机器上运行命令
	//stdout, err := session.StdoutPipe()
	//if err != nil {
	//	log.Fatalf("failed to get stdout: %s", err)
	//}
	//if err := session.Start("ls"); err != nil {
	//	log.Fatalf("failed to start command: %s", err)
	//}
	//
	//output, err := ioutil.ReadAll(stdout)
	//if err != nil {
	//	log.Fatalf("无法读取 stdout：％s", err)
	//}
	//if err := session.Wait(); err != nil {
	//	log.Fatalf("命令失败：％s", err)
	//}
	//fmt.Println(string(output))
}
