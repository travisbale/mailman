package html

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/travisbale/mailman/internal/email"
)

// TemplateDB defines the interface for fetching templates from the database.
type TemplateDB interface {
	GetTemplate(ctx context.Context, name string) (*email.Template, error)
}

// Renderer renders HTML email templates using templates stored in the database.
type Renderer struct {
	db TemplateDB
}

// New creates a new HTML renderer.
func New(db TemplateDB) *Renderer {
	return &Renderer{
		db: db,
	}
}

// Render renders an email template by fetching it from the database and executing it with the provided variables.
func (r *Renderer) Render(ctx context.Context, templateName string, variables map[string]string) (*email.RenderedTemplate, error) {
	// Load template from database
	tmpl, err := r.db.GetTemplate(ctx, templateName)
	if err != nil {
		return nil, err
	}

	// Validate required variables
	if err := validateTemplate(tmpl, variables); err != nil {
		return nil, err
	}

	// Render subject
	subject, err := renderString(tmpl.Subject, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render HTML body (with base template support)
	htmlBody, err := r.renderHTMLWithBase(ctx, tmpl, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML body: %w", err)
	}

	// Render text body if present (with base template support)
	textBody := ""
	if tmpl.TextBody != nil && *tmpl.TextBody != "" {
		textBody, err = r.renderTextWithBase(ctx, tmpl, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render text body: %w", err)
		}
	}

	return &email.RenderedTemplate{
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}, nil
}

// renderHTMLWithBase renders HTML body, loading base templates if needed
func (r *Renderer) renderHTMLWithBase(ctx context.Context, tmpl *email.Template, variables map[string]string) (string, error) {
	// If no base template, render directly
	if tmpl.BaseTemplateName == nil || *tmpl.BaseTemplateName == "" {
		return renderString(tmpl.HTMLBody, variables)
	}

	// Load the entire template chain
	templates, err := r.loadTemplateChain(ctx, tmpl, func(t *email.Template) string {
		return t.HTMLBody
	})
	if err != nil {
		return "", err
	}

	// Parse all templates in the chain
	tmplSet := template.New("base")
	for i := len(templates) - 1; i >= 0; i-- {
		_, err := tmplSet.Parse(templates[i])
		if err != nil {
			return "", fmt.Errorf("failed to parse template in chain: %w", err)
		}
	}

	// Execute the base template (which will call nested {{template "content" .}} blocks)
	var buf bytes.Buffer
	if err := tmplSet.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// renderTextWithBase renders text body, loading base templates if needed
func (r *Renderer) renderTextWithBase(ctx context.Context, tmpl *email.Template, variables map[string]string) (string, error) {
	// If no base template, render directly
	if tmpl.BaseTemplateName == nil || *tmpl.BaseTemplateName == "" {
		return renderString(*tmpl.TextBody, variables)
	}

	// Load the entire template chain
	templates, err := r.loadTemplateChain(ctx, tmpl, func(t *email.Template) string {
		if t.TextBody != nil {
			return *t.TextBody
		}
		return ""
	})
	if err != nil {
		return "", err
	}

	// Parse all templates in the chain
	tmplSet := template.New("base")
	for i := len(templates) - 1; i >= 0; i-- {
		if templates[i] != "" {
			_, err := tmplSet.Parse(templates[i])
			if err != nil {
				return "", fmt.Errorf("failed to parse template in chain: %w", err)
			}
		}
	}

	// Execute the base template
	var buf bytes.Buffer
	if err := tmplSet.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// loadTemplateChain recursively loads all templates in the inheritance chain
// Returns templates in order: [child, parent, grandparent, ...]
func (r *Renderer) loadTemplateChain(ctx context.Context, tmpl *email.Template, extract func(*email.Template) string) ([]string, error) {
	result := []string{extract(tmpl)}

	// Recursively load base templates
	current := tmpl
	seen := make(map[string]bool)
	seen[current.Name] = true

	for current.BaseTemplateName != nil && *current.BaseTemplateName != "" {
		baseName := *current.BaseTemplateName

		// Detect circular references
		if seen[baseName] {
			return nil, fmt.Errorf("circular template reference detected: %s", baseName)
		}
		seen[baseName] = true

		// Load base template
		base, err := r.db.GetTemplate(ctx, baseName)
		if err != nil {
			return nil, fmt.Errorf("failed to load base template %s: %w", baseName, err)
		}

		result = append(result, extract(base))
		current = base
	}

	return result, nil
}

// renderString renders a single string template with variables
func renderString(templateStr string, variables map[string]string) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// validateTemplate checks if all required variables are provided
func validateTemplate(tmpl *email.Template, variables map[string]string) error {
	for _, required := range tmpl.RequiredVariables {
		if _, ok := variables[required]; !ok {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}
	return nil
}
