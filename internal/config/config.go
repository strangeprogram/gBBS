package config

import (
	"encoding/json"
	"fmt"
	"gbbs/internal/irc"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	TelnetPort        int              `json:"telnet_port"`
	SSHPort           int              `json:"ssh_port"`
	WebPort           int              `json:"web_port"`
	GuestbookPath     string           `json:"guestbook_path"`
	WebRoot           string           `json:"web_root"`
	WelcomeScreenPath string           `json:"welcome_screen_path"`
	IRCBridge         irc.BridgeConfig `json:"irc_bridge"`
}

func Load() (*Config, error) {
	cfg := &Config{
		TelnetPort:        2323,
		SSHPort:           2222,
		WebPort:           8080,
		GuestbookPath:     "guestbook.txt",
		WebRoot:           "web",
		WelcomeScreenPath: "welcome.ans",
		IRCBridge: irc.BridgeConfig{
			Enabled: false,
			Port:    6667,
		},
	}

	// Get the executable path
	ex, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v", err)
	}
	exePath := filepath.Dir(ex)

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting current working directory: %v", err)
	}

	configPaths := []string{
		filepath.Join(cwd, "config.json"),
		filepath.Join(cwd, "cmd", "gbbs", "config.json"),
		filepath.Join(exePath, "config.json"),
		filepath.Join(exePath, "cmd", "gbbs", "config.json"),
		"../config.json",
	}

	log.Printf("Searching for config.json in the following locations:")
	for _, path := range configPaths {
		absPath, _ := filepath.Abs(path)
		log.Printf("- %s", absPath)
		_, err := os.Stat(absPath)
		if err == nil {
			log.Printf("  File exists")
		} else if os.IsNotExist(err) {
			log.Printf("  File does not exist")
		} else {
			log.Printf("  Error checking file: %v", err)
		}
	}

	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile == "" {
		return cfg, fmt.Errorf("config file not found in any of the following locations: %v", configPaths)
	}

	log.Printf("Found config file: %s", configFile)

	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(cfg); err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	// Convert relative paths to absolute paths
	cfg.GuestbookPath = makeAbsolute(filepath.Dir(configFile), cfg.GuestbookPath)
	cfg.WebRoot = makeAbsolute(filepath.Dir(configFile), cfg.WebRoot)
	cfg.WelcomeScreenPath = makeAbsolute(filepath.Dir(configFile), cfg.WelcomeScreenPath)

	return cfg, nil
}

func makeAbsolute(basePath, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(basePath, path)
}
