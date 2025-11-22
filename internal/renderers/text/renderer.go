package text

import (
	"context"
	"fmt"
	"strings"

	"github.com/travisbale/mailman/internal/email"
)

// template represents a hardcoded text template
type template struct {
	subject           string
	body              string
	requiredVariables []string
}

// templates contains hardcoded templates for development
var templates = map[string]template{
	"email-verification": {
		subject:           "Verify your email address",
		body:              "Verification URL: {{.verification_url}}",
		requiredVariables: []string{"verification_url"},
	},
	"password-reset": {
		subject:           "Reset your password",
		body:              "Reset URL: {{.reset_url}}",
		requiredVariables: []string{"reset_url"},
	},
}

// Renderer renders simple text-based email templates with hardcoded templates.
type Renderer struct{}

// New creates a new text renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render renders an email template using hardcoded templates and simple variable substitution.
func (r *Renderer) Render(ctx context.Context, templateName string, variables map[string]string) (*email.RenderedTemplate, error) {
	tmpl, exists := templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	for _, required := range tmpl.requiredVariables {
		if _, ok := variables[required]; !ok {
			return nil, fmt.Errorf("missing required variable: %s", required)
		}
	}

	subject := tmpl.subject
	textBody := tmpl.body

	for key, value := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		subject = strings.ReplaceAll(subject, placeholder, value)
		textBody = strings.ReplaceAll(textBody, placeholder, value)
	}

	return &email.RenderedTemplate{
		Subject:  subject,
		HTMLBody: textBody, // Same as text for simplicity
		TextBody: textBody,
	}, nil
}
