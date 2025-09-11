package conf

type Config struct {
	SSHConfigPath string
	Auth
	Web
}

type Auth struct {
	Users map[string]string
}

type Web struct {
	RootPath string
}
