package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

)

type templateDB interface {
	GetTemplate(ctx context.Context, name string) (*Template, error)
	Create(ctx context.Context, template *Template) (*Template, error)
	List(ctx context.Context) ([]*Template, error)
}

// TemplateService handles email template rendering with variable substitution
type TemplateService struct {
	db templateDB
}

// NewTemplateService creates a new template renderer
func NewTemplateService(db templateDB) *TemplateService {
	return &TemplateService{
		db: db,
	}
}

func (s *TemplateService) GetTemplate(ctx context.Context, name string) (*Template, error) {
	return s.db.GetTemplate(ctx, name)
}

// CreateTemplate creates a new template with circular reference validation
func (s *TemplateService) CreateTemplate(ctx context.Context, template *Template) (*Template, error) {
	// Validate no circular references if base template is specified
	if template.BaseTemplateName != nil && *template.BaseTemplateName != "" {
		if err := s.validateNoCircularReference(ctx, template.Name, *template.BaseTemplateName); err != nil {
			return nil, err
		}
	}

	return s.db.Create(ctx, template)
}

// ListTemplates returns all templates
func (s *TemplateService) ListTemplates(ctx context.Context) ([]*Template, error) {
	return s.db.List(ctx)
}

// validateNoCircularReference checks if adding a template would create a circular reference
func (s *TemplateService) validateNoCircularReference(ctx context.Context, newTemplateName, baseTemplateName string) error {
	// Follow the chain of base templates
	seen := make(map[string]bool)
	seen[newTemplateName] = true

	currentName := baseTemplateName
	for currentName != "" {
		// Check if we've seen this template before
		if seen[currentName] {
			return fmt.Errorf("circular reference detected: template '%s' already appears in the inheritance chain", currentName)
		}
		seen[currentName] = true

		// Load the next base template in the chain
		current, err := s.db.GetTemplate(ctx, currentName)
		if err != nil {
			return fmt.Errorf("failed to load base template '%s': %w", currentName, err)
		}

		// Move to the next base template (or empty string if none)
		if current.BaseTemplateName != nil {
			currentName = *current.BaseTemplateName
		} else {
			currentName = ""
		}
	}

	return nil
}

// RenderTemplate renders an email template with the provided variables
// Supports nested templates via base_template_name field
func (s *TemplateService) RenderTemplate(ctx context.Context, tmpl *Template, variables map[string]string) (*RenderedTemplate, error) {
	// Render subject
	subject, err := s.renderString(tmpl.Subject, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render HTML body (with base template support)
	htmlBody, err := s.renderHTMLWithBase(ctx, tmpl, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML body: %w", err)
	}

	// Render text body if present (with base template support)
	textBody := ""
	if tmpl.TextBody != nil && *tmpl.TextBody != "" {
		textBody, err = s.renderTextWithBase(ctx, tmpl, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render text body: %w", err)
		}
	}

	return &RenderedTemplate{
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}, nil
}

// renderHTMLWithBase renders HTML body, loading base templates if needed
func (s *TemplateService) renderHTMLWithBase(ctx context.Context, tmpl *Template, variables map[string]string) (string, error) {
	// If no base template, render directly
	if tmpl.BaseTemplateName == nil || *tmpl.BaseTemplateName == "" {
		return s.renderString(tmpl.HTMLBody, variables)
	}

	// Load the entire template chain
	templates, err := s.loadTemplateChain(ctx, tmpl, func(t *Template) string {
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
func (s *TemplateService) renderTextWithBase(ctx context.Context, tmpl *Template, variables map[string]string) (string, error) {
	// If no base template, render directly
	if tmpl.BaseTemplateName == nil || *tmpl.BaseTemplateName == "" {
		return s.renderString(*tmpl.TextBody, variables)
	}

	// Load the entire template chain
	templates, err := s.loadTemplateChain(ctx, tmpl, func(t *Template) string {
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
func (s *TemplateService) loadTemplateChain(ctx context.Context, tmpl *Template, extract func(*Template) string) ([]string, error) {
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
		base, err := s.db.GetTemplate(ctx, baseName)
		if err != nil {
			return nil, fmt.Errorf("failed to load base template %s: %w", baseName, err)
		}

		result = append(result, extract(base))
		current = base
	}

	return result, nil
}

// renderString renders a single string template with variables
func (*TemplateService) renderString(templateStr string, variables map[string]string) (string, error) {
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

// ValidateTemplate checks if all required variables are provided
func (*TemplateService) ValidateTemplate(tmpl *Template, variables map[string]string) error {
	for _, required := range tmpl.RequiredVariables {
		if _, ok := variables[required]; !ok {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}
	return nil
}
