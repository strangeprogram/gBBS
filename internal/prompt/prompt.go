package prompt

import (
	"fmt"
	"gbbs/internal/config"
	"os"
)

func ReadWelcomeScreen(cfg *config.Config) (string, error) {
	content, err := os.ReadFile(cfg.WelcomeScreenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Welcome to GBBS!\n\nWelcome screen file not found: %s\n", cfg.WelcomeScreenPath), nil
		}
		return "", fmt.Errorf("error reading welcome screen: %v", err)
	}
	return string(content), nil
}
