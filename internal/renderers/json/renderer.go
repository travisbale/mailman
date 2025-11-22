package json

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/travisbale/mailman/internal/email"
)

// Renderer renders email templates as JSON for testing
type Renderer struct{}

// New creates a new JSON renderer.
func New() *Renderer {
	return &Renderer{}
}

// emailData represents the JSON structure of a rendered email
type emailData struct {
	Template  string            `json:"template"`
	Variables map[string]string `json:"variables"`
	Subject   string            `json:"subject"`
}

// Render renders an email template as JSON with the template name and all variables.
func (r *Renderer) Render(ctx context.Context, templateName string, variables map[string]string) (*email.RenderedTemplate, error) {
	data := emailData{
		Template:  templateName,
		Variables: variables,
		Subject:   fmt.Sprintf("[%s]", templateName),
	}

	jsonBody, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email data: %w", err)
	}

	body := string(jsonBody)

	return &email.RenderedTemplate{
		Subject:  data.Subject,
		HTMLBody: body,
		TextBody: body,
	}, nil
}
