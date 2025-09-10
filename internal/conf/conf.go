package conf

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

var (
	Path string // Config path
	Conf Config
)

// LoadConfig Set Path and load config into memory
// Run this at start
func LoadConfig(path string) error {
	Path = path
	err := Update()
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.OpenFile(path, os.O_CREATE, 0644)
			defer f.Close()
			if err == nil {
				return nil
			}
		}
		return fmt.Errorf("failed to load config")
	}
	return nil
}

// Update reads the config file and loads it into the global Conf variable
func Update() (err error) {
	if _, err = os.Stat(Path); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", Path)
	}
	_, err = toml.DecodeFile(Path, &Conf)
	if err != nil {
		return fmt.Errorf("failed to update global config %w", err)
	}
	return nil
}

// Write saves the provided config to the TOML file at the global Path
func Write(conf Config) (err error) {
	f, err := os.Create(Path)
	if err != nil {
		return fmt.Errorf("failed to create config file %w", err)
	}
	defer f.Close()
	err = toml.NewEncoder(f).Encode(conf)
	if err != nil {
		return fmt.Errorf("failed to write config file %w", err)
	}
	return nil
}
