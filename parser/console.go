package parser

import "github.com/travisbale/mailman/internal/clients/console"

// EmailLog represents a parsed email from console logs
type EmailLog = console.EmailLog

// ParseConsoleLogs parses console output and returns structured email logs
func ParseConsoleLogs(logs string) ([]EmailLog, error) {
	return console.ParseLogs(logs)
}
