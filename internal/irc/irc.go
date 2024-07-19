package irc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	ircevent "github.com/thoj/go-ircevent"
)

type BridgeConfig struct {
	Enabled  bool      `json:"enabled"`
	Server   string    `json:"server"`
	Port     int       `json:"port"`
	UseSSL   bool      `json:"use_ssl"`
	Nick     string    `json:"nick"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Bridge struct {
	Config      BridgeConfig
	conn        *ircevent.Connection
	msgChan     chan string
	connected   sync.WaitGroup
	logFile     *os.File
	logMutex    sync.Mutex
	currentDate string
}

func NewBridge(config BridgeConfig) *Bridge {
	bridge := &Bridge{
		Config:      config,
		msgChan:     make(chan string, 1000),
		currentDate: time.Now().Format("2006-01-02"),
	}
	bridge.connected.Add(1)
	go bridge.messageLogger()
	return bridge
}

func (b *Bridge) messageLogger() {
	for msg := range b.msgChan {
		b.logMessage(msg)
	}
}

func (b *Bridge) logMessage(msg string) {
	b.logMutex.Lock()
	defer b.logMutex.Unlock()

	currentDate := time.Now().Format("2006-01-02")
	if currentDate != b.currentDate {
		// Rotate log file
		if b.logFile != nil {
			b.logFile.Close()
		}
		b.currentDate = currentDate
	}

	if b.logFile == nil {
		logDir := "logs"
		os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, fmt.Sprintf("irc_%s.txt", b.currentDate))
		var err error
		b.logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Error opening log file: %v", err)
			return
		}
	}

	_, err := b.logFile.WriteString(msg + "\n")
	if err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}

func (b *Bridge) Connect() error {
	b.conn = ircevent.IRC(b.Config.Nick, b.Config.Nick)
	b.conn.UseTLS = b.Config.UseSSL
	b.conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	b.conn.AddCallback("001", func(e *ircevent.Event) {
		log.Println("Connected to IRC server, joining channels...")
		b.connected.Done()
		b.joinChannels()
	})

	b.conn.AddCallback("JOIN", func(e *ircevent.Event) {
		channel := e.Arguments[0]
		log.Printf("Joined channel: %s", channel)
	})

	b.conn.AddCallback("KICK", func(e *ircevent.Event) {
		channel := e.Arguments[0]
		kicked := e.Arguments[1]
		if kicked == b.Config.Nick {
			log.Printf("Kicked from %s, attempting to rejoin in 3 seconds...", channel)
			time.Sleep(3 * time.Second)
			b.conn.Join(channel)
		}
	})

	b.conn.AddCallback("PRIVMSG", func(e *ircevent.Event) {
		msg := fmt.Sprintf("%s <%s> %s: %s",
			time.Now().Format("2006-01-02 15:04:05"),
			e.Arguments[0], e.Nick, sanitizeMessage(e.Message()))
		select {
		case b.msgChan <- msg:
		default:
			log.Printf("Message channel full, dropping message: %s", msg)
		}
	})

	b.conn.AddCallback("PING", func(e *ircevent.Event) {
		b.conn.SendRaw("PONG :" + e.Message())
	})

	err := b.conn.Connect(fmt.Sprintf("%s:%d", b.Config.Server, b.Config.Port))
	if err != nil {
		return err
	}

	go b.conn.Loop()

	b.connected.Wait()

	return nil
}

func (b *Bridge) joinChannels() {
	for _, channel := range b.Config.Channels {
		if channel.Password != "" {
			b.conn.Join(channel.Name + " " + channel.Password)
		} else {
			b.conn.Join(channel.Name)
		}
	}
}

func (b *Bridge) SendMessage(channel, sender, message string) {
	b.conn.Privmsg(channel, fmt.Sprintf("<%s> %s", sender, message))
}

func (b *Bridge) GetMessageChannel() <-chan string {
	return b.msgChan
}

func (b *Bridge) GetRecentMessages(count int) ([]string, error) {
	b.logMutex.Lock()
	defer b.logMutex.Unlock()

	logPath := filepath.Join("logs", fmt.Sprintf("irc_%s.txt", b.currentDate))
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		messages = append(messages, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if count > len(messages) {
		count = len(messages)
	}
	return messages[len(messages)-count:], nil
}

func sanitizeMessage(message string) string {
	if !utf8.ValidString(message) {
		return strings.ToValidUTF8(message, "\uFFFD")
	}
	return message
}

func (b *Bridge) Close() {
	if b.conn != nil {
		b.conn.Quit()
	}
	close(b.msgChan)
	if b.logFile != nil {
		b.logFile.Close()
	}
}
