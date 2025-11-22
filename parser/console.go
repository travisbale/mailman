package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EmailLog represents a parsed email from console logs
type EmailLog struct {
	To        string            `json:"to"`        // Recipient email address
	Template  string            `json:"template"`  // Template name used
	Variables map[string]string `json:"variables"` // Template variables (includes URLs with tokens)
	Subject   string            `json:"subject"`   // Rendered subject line
}

// ParseConsoleLogs parses console output and returns structured email logs
// This function knows the exact format produced by the console client
func ParseConsoleLogs(logs string) ([]EmailLog, error) {
	var emails []EmailLog
	lines := strings.Split(logs, "\n")

	var inTextBody bool
	var jsonLines []string
	var currentRecipient string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Extract recipient from "To:" line
		if recipientEmail, ok := strings.CutPrefix(trimmed, "To: "); ok {
			currentRecipient = recipientEmail
		}

		// Entering the Text Body section (contains JSON)
		if trimmed == "Text Body:" {
			inTextBody = true
			jsonLines = []string{}
			continue
		}

		// Exiting the Text Body section
		if inTextBody && strings.HasPrefix(trimmed, "---") {
			if err := parseEmailJSON(currentRecipient, jsonLines, &emails); err != nil {
				return nil, err
			}

			// Reset state for next email
			inTextBody = false
			jsonLines = []string{}
			currentRecipient = ""
			continue
		}

		// Accumulate JSON lines from text body
		if inTextBody {
			jsonLines = append(jsonLines, trimmed)
		}
	}

	return emails, nil
}

// parseEmailJSON unmarshals the JSON from text body and appends to emails list
func parseEmailJSON(recipient string, jsonLines []string, emails *[]EmailLog) error {
	if len(jsonLines) == 0 {
		return nil
	}

	jsonStr := strings.Join(jsonLines, "\n")

	var log EmailLog
	if err := json.Unmarshal([]byte(jsonStr), &log); err != nil {
		return fmt.Errorf("failed to unmarshal email JSON: %w", err)
	}

	// Add recipient from the "To:" line
	log.To = recipient

	*emails = append(*emails, log)
	return nil
}
