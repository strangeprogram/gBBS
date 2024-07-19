package telnet

import (
	"bufio"
	"fmt"
	"gbbs/internal/config"
	"gbbs/internal/irc"
	"gbbs/internal/messageboard"
	"gbbs/internal/user"
	"net"
	"os"
	"strings"
	"time"
)

func Serve(cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.TelnetPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, cfg, userManager, messageBoard, ircBridge)
	}
}

func handleConnection(conn net.Conn, cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) {
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	welcomeScreen, err := os.ReadFile(cfg.WelcomeScreenPath)
	if err != nil {
		fmt.Fprintf(writer, "Error reading welcome screen: %v\n", err)
		return
	}

	fmt.Fprint(writer, string(welcomeScreen))
	fmt.Fprint(writer, "\n\n\n\n\n") // Add five newlines after the welcome screen
	writer.Flush()

	for {
		fmt.Fprintf(writer, "\033[0;32mChoose (L)ogin or (R)egister: \033[0m")
		writer.Flush()

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch strings.ToLower(choice) {
		case "l":
			username, err := login(reader, writer, userManager)
			if err != nil {
				fmt.Fprintf(writer, "\033[0;31mLogin failed: %v\033[0m\n", err)
				writer.Flush()
				continue
			}
			fmt.Fprintf(writer, "\n\033[1;32mLogin successful! Welcome, %s!\033[0m\n", username)
			writer.Flush()
			time.Sleep(2 * time.Second)
			handleBBS(username, reader, writer, messageBoard, ircBridge)
			return
		case "r":
			username, err := register(reader, writer, userManager)
			if err != nil {
				fmt.Fprintf(writer, "\033[0;31mRegistration failed: %v\033[0m\n", err)
				writer.Flush()
				continue
			}
			fmt.Fprintf(writer, "\n\033[1;32mRegistration successful! Welcome, %s!\033[0m\n", username)
			writer.Flush()
			time.Sleep(2 * time.Second)
			handleBBS(username, reader, writer, messageBoard, ircBridge)
			return
		default:
			fmt.Fprintf(writer, "\033[0;31mInvalid choice. Please enter 'L' or 'R'.\033[0m\n")
			writer.Flush()
		}
	}
}

func login(reader *bufio.Reader, writer *bufio.Writer, userManager *user.Manager) (string, error) {
	fmt.Fprintf(writer, "Username: ")
	writer.Flush()
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Fprintf(writer, "Password: ")
	writer.Flush()
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	authenticated, err := userManager.Authenticate(username, password)
	if err != nil {
		return "", err
	}
	if !authenticated {
		return "", fmt.Errorf("invalid username or password")
	}

	return username, nil
}

func register(reader *bufio.Reader, writer *bufio.Writer, userManager *user.Manager) (string, error) {
	fmt.Fprintf(writer, "Choose a username: ")
	writer.Flush()
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Fprintf(writer, "Choose a password: ")
	writer.Flush()
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	err := userManager.CreateUser(username, password)
	if err != nil {
		return "", err
	}

	return username, nil
}

func handleBBS(username string, reader *bufio.Reader, writer *bufio.Writer, messageBoard *messageboard.MessageBoard, ircBridge *irc.Bridge) {
	for {
		fmt.Fprintf(writer, "\n\033[0;36mBBS Menu:\033[0m\n")
		fmt.Fprintf(writer, "1. Read messages\n")
		fmt.Fprintf(writer, "2. Post message\n")
		fmt.Fprintf(writer, "3. IRC Bridge\n")
		fmt.Fprintf(writer, "4. Logout\n")
		fmt.Fprintf(writer, "Choice: ")
		writer.Flush()

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			messages, err := messageBoard.GetMessages()
			if err != nil {
				fmt.Fprintf(writer, "\033[0;31mError reading messages: %v\033[0m\n", err)
			} else {
				for _, msg := range messages {
					fmt.Fprintf(writer, "%s\n", msg)
				}
			}
		case "2":
			fmt.Fprintf(writer, "Enter your message: ")
			writer.Flush()
			message, _ := reader.ReadString('\n')
			message = strings.TrimSpace(message)
			err := messageBoard.PostMessage(username, message)
			if err != nil {
				fmt.Fprintf(writer, "\033[0;31mError posting message: %v\033[0m\n", err)
			} else {
				fmt.Fprintf(writer, "\033[0;32mMessage posted successfully!\033[0m\n")
			}
		case "3":
			handleIRCBridge(username, reader, writer, ircBridge)
		case "4":
			fmt.Fprintf(writer, "\033[0;33mGoodbye!\033[0m\n")
			writer.Flush()
			return
		default:
			fmt.Fprintf(writer, "\033[0;31mInvalid choice. Please try again.\033[0m\n")
		}
		writer.Flush()
	}
}

func handleIRCBridge(username string, reader *bufio.Reader, writer *bufio.Writer, ircBridge *irc.Bridge) {
	if ircBridge == nil {
		fmt.Fprintf(writer, "\033[0;31mIRC Bridge is not enabled.\033[0m\n")
		writer.Flush()
		return
	}

	fmt.Fprintf(writer, "\033[0;36mEntering IRC Bridge mode. Type '/quit' to exit.\033[0m\n")
	writer.Flush()

	// Fetch recent messages
	recentMessages, err := ircBridge.GetRecentMessages(50) // Get last 50 messages
	if err != nil {
		fmt.Fprintf(writer, "\033[0;31mError fetching recent messages: %v\033[0m\n", err)
	} else {
		for _, msg := range recentMessages {
			fmt.Fprintf(writer, "%s\n", msg)
		}
	}
	writer.Flush()

	ircMsgChan := ircBridge.GetMessageChannel()

	for {
		fmt.Fprintf(writer, "> ")
		writer.Flush()

		// Check for new IRC messages
		select {
		case msg := <-ircMsgChan:
			fmt.Fprintf(writer, "\r%s\n> ", msg)
			writer.Flush()
		default:
			// No new messages, continue to read user input
		}

		// Read user input
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "/quit" {
			fmt.Fprintf(writer, "\033[0;36mExiting IRC Bridge mode.\033[0m\n")
			writer.Flush()
			return
		}

		// Send user message to IRC
		for _, channel := range ircBridge.Config.Channels {
			ircBridge.SendMessage(channel.Name, username, input)
		}
	}
}
