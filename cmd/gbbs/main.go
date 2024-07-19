package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"gbbs/internal/config"
	"gbbs/internal/irc"
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

	// Set the working directory to the directory containing the executable
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	exePath := filepath.Dir(ex)
	err = os.Chdir(exePath)
	if err != nil {
		log.Fatalf("Failed to change working directory: %v", err)
	}

	log.Printf("Working directory: %s", exePath)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	var ircBridge *irc.Bridge
	if cfg.IRCBridge.Enabled {
		ircBridge = irc.NewBridge(cfg.IRCBridge)
		err := ircBridge.Connect()
		if err != nil {
			log.Printf("Failed to connect to IRC: %v", err)
		} else {
			log.Printf("Connected to IRC server %s", cfg.IRCBridge.Server)
		}
	}

	// Set up graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		if err := telnet.Serve(cfg, userManager, messageBoard, ircBridge); err != nil {
			log.Printf("Telnet server error: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := ssh.Serve(cfg, userManager, messageBoard, ircBridge); err != nil {
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

	// Wait for shutdown signal
	<-shutdown
	log.Println("Shutting down...")

	// Close IRC bridge
	if ircBridge != nil {
		ircBridge.Close()
	}

	// TODO: Add code to gracefully shut down telnet, ssh, and web servers

	wg.Wait()
	log.Println("Shutdown complete")
}
