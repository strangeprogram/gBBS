package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"gbbs/internal/config"
	"gbbs/internal/messageboard"
	"gbbs/internal/prompt"
	"gbbs/internal/user"
)

func generateSSHKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func Serve(cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard) error {
	signer, err := generateSSHKey()
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %v", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.SSHPort))
	if err != nil {
		return err
	}
	log.Printf("SSH server listening on port %d", cfg.SSHPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %v", err)
			continue
		}
		go handleConnection(conn, config, cfg, userManager, messageBoard)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig, cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

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
				ok := false
				switch req.Type {
				case "shell":
					ok = true
					if len(req.Payload) > 0 {
						ok = false
					}
				case "pty-req":
					ok = true
				}
				req.Reply(ok, nil)
			}
		}(requests)

		term := term.NewTerminal(channel, "> ")
		go handleSSHSession(term, cfg, userManager, messageBoard)
	}
}

func handleSSHSession(term *term.Terminal, cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard) {
	defer term.Write([]byte("Goodbye!\n"))

	welcomeScreen, err := prompt.ReadWelcomeScreen(cfg)
	if err != nil {
		fmt.Fprintf(term, "Error reading welcome screen: %v\n", err)
		return
	}

	term.Write([]byte(welcomeScreen))
	term.Write([]byte("\n\n\n\n\n")) // new lines to make it look real nice yo

	for {
		term.SetPrompt("\033[0;32mChoose (L)ogin or (R)egister: \033[0m")
		choice, err := term.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(term, "\033[0;31mError reading input: %v\033[0m\n", err)
			continue
		}

		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "l":
			username, err := login(term, userManager)
			if err != nil {
				fmt.Fprintf(term, "\033[0;31mLogin failed: %v\033[0m\n", err)
				continue
			}
			fmt.Fprintf(term, "\n\033[1;32mLogin successful! Welcome, %s!\033[0m\n", username)
			time.Sleep(2 * time.Second)
			handleBBS(term, username, messageBoard)
			return
		case "r":
			username, err := register(term, userManager)
			if err != nil {
				fmt.Fprintf(term, "\033[0;31mRegistration failed: %v\033[0m\n", err)
				continue
			}
			fmt.Fprintf(term, "\n\033[1;32mRegistration successful! Welcome, %s!\033[0m\n", username)
			time.Sleep(2 * time.Second)
			handleBBS(term, username, messageBoard)
			return
		default:
			fmt.Fprintf(term, "\033[0;31mInvalid choice. Please enter 'L' or 'R'.\033[0m\n")
		}
	}
}

func login(term *term.Terminal, userManager *user.Manager) (string, error) {
	term.SetPrompt("Username: ")
	username, err := term.ReadLine()
	if err != nil {
		return "", err
	}

	term.SetPrompt("Password: ")
	password, err := term.ReadPassword("Password: ")
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

func register(term *term.Terminal, userManager *user.Manager) (string, error) {
	term.SetPrompt("Choose a username: ")
	username, err := term.ReadLine()
	if err != nil {
		return "", err
	}

	term.SetPrompt("Choose a password: ")
	password, err := term.ReadPassword("Choose a password: ")
	if err != nil {
		return "", err
	}

	err = userManager.CreateUser(username, password)
	if err != nil {
		return "", err
	}

	return username, nil
}

func handleBBS(term *term.Terminal, username string, messageBoard *messageboard.MessageBoard) {
	for {
		term.Write([]byte("\n\033[0;36mBBS Menu:\033[0m\n"))
		term.Write([]byte("1. Read messages\n"))
		term.Write([]byte("2. Post message\n"))
		term.Write([]byte("3. Logout\n"))
		term.SetPrompt("Choice: ")

		choice, err := term.ReadLine()
		if err != nil {
			fmt.Fprintf(term, "\033[0;31mError reading input: %v\033[0m\n", err)
			continue
		}

		switch strings.TrimSpace(choice) {
		case "1":
			messages, err := messageBoard.GetMessages()
			if err != nil {
				fmt.Fprintf(term, "\033[0;31mError reading messages: %v\033[0m\n", err)
			} else {
				for _, msg := range messages {
					fmt.Fprintf(term, "%s\n", msg)
				}
			}
		case "2":
			term.SetPrompt("Enter your message: ")
			message, err := term.ReadLine()
			if err != nil {
				fmt.Fprintf(term, "\033[0;31mError reading message: %v\033[0m\n", err)
				continue
			}
			err = messageBoard.PostMessage(username, message)
			if err != nil {
				fmt.Fprintf(term, "\033[0;31mError posting message: %v\033[0m\n", err)
			} else {
				fmt.Fprintf(term, "\033[0;32mMessage posted successfully!\033[0m\n")
			}
		case "3":
			return
		default:
			fmt.Fprintf(term, "\033[0;31mInvalid choice. Please try again.\033[0m\n")
		}
	}
}
