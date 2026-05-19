package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

type Config struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

var cfg *Config

func configPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.recon/config.json"
}

func LoadConfig() error {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}
	cfg = &c
	return nil
}

func CheckPassword(password string) bool {
	if cfg == nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(password)) == nil
}

func RunSetup() error {
	home, _ := os.UserHomeDir()
	for _, d := range []string{
		home + "/.recon",
		home + "/.recon/databases",
		home + "/.recon/logs",
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", d, err)
		}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	fmt.Print("Password: ")
	passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password := strings.TrimSpace(string(passBytes))
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	c := Config{
		Username:     username,
		PasswordHash: string(hash),
	}

	data, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("[+] Config saved to", configPath())
	return nil
}
