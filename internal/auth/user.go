package auth

import (
	"fmt"
	"minimalpanel/internal/conf"

	"golang.org/x/crypto/bcrypt"
)

// NewUser creates a new user with hashed password and saves it to the config file
func NewUser(name string, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	newConf := conf.Conf

	if newConf.Auth.Users == nil {
		newConf.Auth.Users = make(map[string]string)
	}
	newConf.Auth.Users[name] = string(hash)

	err = conf.Write(newConf)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	conf.Conf = newConf

	return nil
}

// VerifyPassword verifies a user's password against the stored hash
func VerifyPassword(name string, password string) bool {
	hashedPassword, exists := conf.Conf.Auth.Users[name]
	if !exists {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
