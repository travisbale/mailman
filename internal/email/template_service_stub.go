package email

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StubTemplateService provides mock templates for development without requiring database
type StubTemplateService struct{}

// NewStubTemplateService creates a new stub template service for development
func NewStubTemplateService() *StubTemplateService {
	return &StubTemplateService{}
}

// GetTemplate returns a mock template based on the template name
func (s *StubTemplateService) GetTemplate(ctx context.Context, name string) (*Template, error) {
	template, exists := stubTemplates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return &template, nil
}

// RenderTemplate renders a template by doing simple variable substitution
func (s *StubTemplateService) RenderTemplate(ctx context.Context, tmpl *Template, variables map[string]string) (*RenderedTemplate, error) {
	// Validate first
	if err := s.ValidateTemplate(tmpl, variables); err != nil {
		return nil, err
	}

	// Simple string replacement for variables
	subject := tmpl.Subject
	htmlBody := tmpl.HTMLBody
	textBody := ""

	for key, value := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		subject = strings.ReplaceAll(subject, placeholder, value)
		htmlBody = strings.ReplaceAll(htmlBody, placeholder, value)
	}

	if tmpl.TextBody != nil {
		textBody = *tmpl.TextBody
		for key, value := range variables {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			textBody = strings.ReplaceAll(textBody, placeholder, value)
		}
	}

	return &RenderedTemplate{
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}, nil
}

// ValidateTemplate checks that all required variables are provided
func (s *StubTemplateService) ValidateTemplate(tmpl *Template, variables map[string]string) error {
	for _, required := range tmpl.RequiredVariables {
		if _, ok := variables[required]; !ok {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}
	return nil
}

// stubTemplates contains hardcoded templates for development
var stubTemplates = map[string]Template{
	"email-verification": {
		Name:              "email-verification",
		Subject:           "Verify your email address",
		HTMLBody:          "Email: {{.email}}\nVerification URL: {{.verification_url}}",
		TextBody:          stringPtr("Email: {{.email}}\nVerification URL: {{.verification_url}}"),
		RequiredVariables: []string{"email", "verification_url"},
		Version:           1,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	},
	"password-reset": {
		Name:              "password-reset",
		Subject:           "Reset your password",
		HTMLBody:          "Email: {{.email}}\nReset URL: {{.reset_url}}",
		TextBody:          stringPtr("Email: {{.email}}\nReset URL: {{.reset_url}}"),
		RequiredVariables: []string{"email", "reset_url"},
		Version:           1,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	},
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}
