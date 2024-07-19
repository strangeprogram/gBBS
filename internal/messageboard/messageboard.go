package messageboard

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

type MessageBoard struct {
	filePath string
	mu       sync.Mutex
}

func New(filePath string) (*MessageBoard, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	file.Close()

	return &MessageBoard{filePath: filePath}, nil
}

func (mb *MessageBoard) PostMessage(username, message string) error {
	if len(message) == 0 {
		return fmt.Errorf("message cannot be empty")
	}
	if len(message) > 500 {
		return fmt.Errorf("message too long (max 500 characters)")
	}

	mb.mu.Lock()
	defer mb.mu.Unlock()

	file, err := os.OpenFile(mb.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02 15:04:05"), username, message))
	return err
}

func (mb *MessageBoard) GetMessages() ([]string, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	file, err := os.Open(mb.filePath)
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

	return messages, nil
}
