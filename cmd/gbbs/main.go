package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gbbs/internal/config"
	"gbbs/internal/messageboard"
	"gbbs/internal/ssh"
	"gbbs/internal/telnet"
	"gbbs/internal/user"
	"gbbs/internal/web"
)

var debug = flag.Bool("debug", false, "Enable debug mode")

func main() {
	flag.Parse()

	if *debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetOutput(os.NewFile(0, os.DevNull))
	}

	// Get the executable path
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	exePath := filepath.Dir(ex)

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	log.Printf("Executable directory: %s", exePath)
	log.Printf("Current working directory: %s", cwd)

	// Try to load config from multiple locations
	configPaths := []string{
		filepath.Join(cwd, "config.json"),
		filepath.Join(cwd, "cmd", "gbbs", "config.json"),
		filepath.Join(exePath, "config.json"),
		filepath.Join(exePath, "cmd", "gbbs", "config.json"),
	}

	var cfg *config.Config
	var configErr error
	for _, path := range configPaths {
		cfg, configErr = config.Load(path)
		if configErr == nil {
			log.Printf("Loaded configuration from: %s", path)
			break
		}
	}

	if configErr != nil {
		log.Fatalf("Failed to load configuration: %v\nTried paths: %v", configErr, configPaths)
	}

	log.Printf("Loaded configuration: %+v", cfg)

	userManager, err := user.NewManager("bbs.db")
	if err != nil {
		log.Fatalf("Failed to initialize user manager: %v", err)
	}
	defer userManager.Close()

	messageBoard, err := messageboard.New(cfg.GuestbookPath)
	if err != nil {
		log.Fatalf("Failed to initialize message board: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		if err := telnet.Serve(cfg, userManager, messageBoard); err != nil {
			log.Printf("Telnet server error: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := ssh.Serve(cfg, userManager, messageBoard); err != nil {
			log.Printf("SSH server error: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := web.Serve(cfg.WebPort, cfg.WebRoot, userManager, messageBoard); err != nil {
			log.Printf("Web server error: %v", err)
		}
	}()

	log.Printf("BBS is running. Telnet: %d, SSH: %d, Web: %d", cfg.TelnetPort, cfg.SSHPort, cfg.WebPort)

	wg.Wait()
}
