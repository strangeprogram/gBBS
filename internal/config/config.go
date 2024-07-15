package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	TelnetPort        int    `json:"telnet_port"`
	SSHPort           int    `json:"ssh_port"`
	WebPort           int    `json:"web_port"`
	GuestbookPath     string `json:"guestbook_path"`
	WebRoot           string `json:"web_root"`
	WelcomeScreenPath string `json:"welcome_screen_path"`
}

func Load(configPath string) (*Config, error) {
	cfg := &Config{
		TelnetPort:        2323,
		SSHPort:           2222,
		WebPort:           8080,
		GuestbookPath:     "guestbook.txt",
		WebRoot:           "web",
		WelcomeScreenPath: "welcome.ans",
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(cfg); err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	// Convert relative paths to absolute paths
	cfg.GuestbookPath = makeAbsolute(filepath.Dir(configPath), cfg.GuestbookPath)
	cfg.WebRoot = makeAbsolute(filepath.Dir(configPath), cfg.WebRoot)
	cfg.WelcomeScreenPath = makeAbsolute(filepath.Dir(configPath), cfg.WelcomeScreenPath)

	return cfg, nil
}

func makeAbsolute(basePath, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(basePath, path)
}
