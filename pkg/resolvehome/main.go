package resolvehome

import (
	"fmt"
	"os"
	"strings"
)

var (
	homes = []string{"$HOME", "${HOME}", "~"}
)

func Resolve(s string) (string, error) {
	for _, home := range homes {
		if strings.Contains(s, home) {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("determining current user: %w", err)
			}
			s = strings.Replace(s, home, homeDir, -1)
		}
	}

	return s, nil
}
