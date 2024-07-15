package telnet

import (
	"bufio"
	"fmt"
	"gbbs/internal/config"
	"gbbs/internal/messageboard"
	"gbbs/internal/prompt"
	"gbbs/internal/user"
	"net"
	"strings"
	"time"
)

func Serve(cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard) error {
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
		go handleConnection(conn, cfg, userManager, messageBoard)
	}
}

func handleConnection(conn net.Conn, cfg *config.Config, userManager *user.Manager, messageBoard *messageboard.MessageBoard) {
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	welcomeScreen, err := prompt.ReadWelcomeScreen(cfg)
	if err != nil {
		fmt.Fprintf(writer, "Error reading welcome screen: %v\n", err)
		return
	}

	fmt.Fprint(writer, welcomeScreen)
	writer.Flush()

	for {
		fmt.Fprintf(writer, "\n\033[0;31mPlease choose an option:\033[0m\n")
		fmt.Fprintf(writer, "\033[1;32mL\033[0m - Login\n")
		fmt.Fprintf(writer, "\033[1;32mR\033[0m - Register\n\n")
		fmt.Fprintf(writer, "\033[0;32mYour choice: \033[0m")
		writer.Flush()

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if strings.EqualFold(choice, "L") {
			username, err := login(reader, writer, userManager)
			if err != nil {
				fmt.Fprintf(writer, "\033[0;31mLogin failed: %v\033[0m\n", err)
				writer.Flush()
				continue
			}
			fmt.Fprintf(writer, "\n\033[1;32mLogin successful! Welcome, %s!\033[0m\n", username)
			writer.Flush()
			time.Sleep(2 * time.Second) // Pause for 2 seconds to show the message
			handleBBS(username, reader, writer, messageBoard)
			return
		} else if strings.EqualFold(choice, "R") {
			username, err := register(reader, writer, userManager)
			if err != nil {
				fmt.Fprintf(writer, "\033[0;31mRegistration failed: %v\033[0m\n", err)
				writer.Flush()
				continue
			}
			fmt.Fprintf(writer, "\n\033[1;32mRegistration successful! Welcome, %s!\033[0m\n", username)
			writer.Flush()
			time.Sleep(2 * time.Second) // Pause for 2 seconds to show the message
			handleBBS(username, reader, writer, messageBoard)
			return
		} else {
			fmt.Fprintf(writer, "\033[0;31mInvalid choice. Please try again.\033[0m\n")
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

func handleBBS(username string, reader *bufio.Reader, writer *bufio.Writer, messageBoard *messageboard.MessageBoard) {
	for {
		fmt.Fprintf(writer, "\n\033[0;36mBBS Menu:\033[0m\n")
		fmt.Fprintf(writer, "1. Read messages\n")
		fmt.Fprintf(writer, "2. Post message\n")
		fmt.Fprintf(writer, "3. Logout\n")
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
			fmt.Fprintf(writer, "\033[0;33mGoodbye!\033[0m\n")
			writer.Flush()
			return
		default:
			fmt.Fprintf(writer, "\033[0;31mInvalid choice. Please try again.\033[0m\n")
		}
		writer.Flush()
	}
}
