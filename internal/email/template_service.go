package email

import (
	"context"
	"fmt"
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
