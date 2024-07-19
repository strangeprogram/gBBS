package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"gbbs/internal/config"
	"gbbs/internal/irc"
	"gbbs/internal/messageboard"
	"gbbs/internal/user"
)

func generateSSHKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func Serve(cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) error {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	private, err := generateSSHKey()
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %v", err)
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.SSHPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %v", err)
			continue
		}
		go handleConnection(conn, config, cfg, userManager, messageBoard, ircBridge)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig, cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "shell":
					req.Reply(true, nil)
				case "pty-req":
					req.Reply(true, nil)
				}
			}
		}(requests)

		terminal := term.NewTerminal(channel, "")
		go handleSSHSession(terminal, cfg, userManager, messageBoard, ircBridge)
	}
}

func handleSSHSession(terminal *term.Terminal, cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) {
	defer terminal.Write([]byte("Goodbye!\n"))

	welcomeScreen, err := os.ReadFile(cfg.WelcomeScreenPath)
	if err != nil {
		log.Printf("Error reading welcome screen: %v", err)
		terminal.Write([]byte("Welcome to GBBS!\n\n"))
	} else {
		terminal.Write(welcomeScreen)
	}
	terminal.Write([]byte("\n\n\n\n\n")) // Add five newlines after the welcome screen

	for {
		terminal.SetPrompt("\033[0;32mChoose (L)ogin or (R)egister: \033[0m")
		choice, err := terminal.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("Error reading input: %v", err)
			continue
		}

		switch strings.ToLower(choice) {
		case "l":
			username, err := login(terminal, userManager)
			if err != nil {
				terminal.Write([]byte(fmt.Sprintf("\033[0;31mLogin failed: %v\033[0m\n", err)))
				continue
			}
			terminal.Write([]byte(fmt.Sprintf("\n\033[1;32mLogin successful! Welcome, %s!\033[0m\n", username)))
			time.Sleep(2 * time.Second)
			handleBBS(username, terminal, messageBoard, ircBridge)
			return
		case "r":
			username, err := register(terminal, userManager)
			if err != nil {
				terminal.Write([]byte(fmt.Sprintf("\033[0;31mRegistration failed: %v\033[0m\n", err)))
				continue
			}
			terminal.Write([]byte(fmt.Sprintf("\n\033[1;32mRegistration successful! Welcome, %s!\033[0m\n", username)))
			time.Sleep(2 * time.Second)
			handleBBS(username, terminal, messageBoard, ircBridge)
			return
		default:
			terminal.Write([]byte("\033[0;31mInvalid choice. Please enter 'L' or 'R'.\033[0m\n"))
		}
	}
}

func login(terminal *term.Terminal, userManager *user.Manager) (string, error) {
	terminal.SetPrompt("Username: ")
	username, err := terminal.ReadLine()
	if err != nil {
		return "", err
	}

	terminal.SetPrompt("Password: ")
	password, err := terminal.ReadPassword("Password: ")
	if err != nil {
		return "", err
	}

	authenticated, err := userManager.Authenticate(username, password)
	if err != nil {
		return "", err
	}
	if !authenticated {
		return "", fmt.Errorf("invalid username or password")
	}

	return username, nil
}

func register(terminal *term.Terminal, userManager *user.Manager) (string, error) {
	terminal.SetPrompt("Choose a username: ")
	username, err := terminal.ReadLine()
	if err != nil {
		return "", err
	}

	terminal.SetPrompt("Choose a password: ")
	password, err := terminal.ReadPassword("Choose a password: ")
	if err != nil {
		return "", err
	}

	err = userManager.CreateUser(username, password)
	if err != nil {
		return "", err
	}

	return username, nil
}

func handleBBS(username string, terminal *term.Terminal, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) {
	for {
		terminal.Write([]byte("\n\033[0;36mBBS Menu:\033[0m\n"))
		terminal.Write([]byte("1. Read messages\n"))
		terminal.Write([]byte("2. Post message\n"))
		terminal.Write([]byte("3. IRC Bridge\n"))
		terminal.Write([]byte("4. Logout\n"))
		terminal.SetPrompt("Choice: ")

		choice, err := terminal.ReadLine()
		if err != nil {
			log.Printf("Error reading input: %v", err)
			continue
		}

		switch strings.TrimSpace(choice) {
		case "1":
			messages, err := messageBoard.GetMessages()
			if err != nil {
				terminal.Write([]byte(fmt.Sprintf("\033[0;31mError reading messages: %v\033[0m\n", err)))
			} else {
				for _, msg := range messages {
					terminal.Write([]byte(fmt.Sprintf("%s\n", msg)))
				}
			}
		case "2":
			terminal.SetPrompt("Enter your message: ")
			message, err := terminal.ReadLine()
			if err != nil {
				terminal.Write([]byte(fmt.Sprintf("\033[0;31mError reading message: %v\033[0m\n", err)))
				continue
			}
			err = messageBoard.PostMessage(username, message)
			if err != nil {
				terminal.Write([]byte(fmt.Sprintf("\033[0;31mError posting message: %v\033[0m\n", err)))
			} else {
				terminal.Write([]byte("\033[0;32mMessage posted successfully!\033[0m\n"))
			}
		case "3":
			handleIRCBridge(username, terminal, ircBridge)
		case "4":
			return
		default:
			terminal.Write([]byte("\033[0;31mInvalid choice. Please try again.\033[0m\n"))
		}
	}
}

func handleIRCBridge(username string, terminal *term.Terminal, ircBridge *irc.Bridge) {
	if ircBridge == nil {
		terminal.Write([]byte("\033[0;31mIRC Bridge is not enabled.\033[0m\n"))
		return
	}

	terminal.Write([]byte("\033[0;36mEntering IRC Bridge mode. Type '/quit' to exit.\033[0m\n"))

	// Fetch recent messages
	recentMessages, err := ircBridge.GetRecentMessages(50) // Get last 50 messages
	if err != nil {
		terminal.Write([]byte(fmt.Sprintf("\033[0;31mError fetching recent messages: %v\033[0m\n", err)))
	} else {
		for _, msg := range recentMessages {
			terminal.Write([]byte(fmt.Sprintf("%s\n", msg)))
		}
	}

	ircMsgChan := ircBridge.GetMessageChannel()
	quit := make(chan struct{})

	// Goroutine to handle incoming IRC messages
	go func() {
		for {
			select {
			case msg := <-ircMsgChan:
				terminal.Write([]byte(fmt.Sprintf("\r%s\n> ", msg)))
			case <-quit:
				return
			}
		}
	}()

	// Main loop for user input
	for {
		terminal.SetPrompt("> ")
		input, err := terminal.ReadLine()
		if err != nil {
			if err == io.EOF {
				close(quit)
				return
			}
			log.Printf("Error reading input: %v", err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "/quit" {
			close(quit)
			terminal.Write([]byte("\033[0;36mExiting IRC Bridge mode.\033[0m\n"))
			return
		}

		// Send user message to IRC
		for _, channel := range ircBridge.Config.Channels {
			ircBridge.SendMessage(channel.Name, username, input)
		}
	}
}
