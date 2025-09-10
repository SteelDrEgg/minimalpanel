package conf

type Config struct {
	SSHConfigPath string
	Auth
}

type Auth struct {
	Users map[string]string
}
