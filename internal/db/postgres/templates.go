package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/travisbale/mailman/internal/db/postgres/internal/sqlc"
	"github.com/travisbale/mailman/internal/email"
)

// TemplatesDB handles database operations for email templates
type TemplatesDB struct {
	db *DB
}

// NewTemplatesDB creates a new templates database adapter
func NewTemplatesDB(db *DB) *TemplatesDB {
	return &TemplatesDB{db: db}
}

// GetTemplate retrieves a template by its name
func (r *TemplatesDB) GetTemplate(ctx context.Context, name string) (*email.Template, error) {
	queries := sqlc.New(r.db.Pool())

	dbTemplate, err := queries.GetTemplate(ctx, name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("template not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return convertTemplateToDomain(dbTemplate), nil
}

// List retrieves all email templates, optionally filtered by name
func (r *TemplatesDB) List(ctx context.Context) ([]*email.Template, error) {
	queries := sqlc.New(r.db.Pool())

	dbTemplates, err := queries.ListTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	templates := make([]*email.Template, len(dbTemplates))
	for i := range dbTemplates {
		templates[i] = convertTemplateToDomain(dbTemplates[i])
	}

	return templates, nil
}

// Create inserts a new email template
func (r *TemplatesDB) Create(ctx context.Context, template *email.Template) (*email.Template, error) {
	queries := sqlc.New(r.db.Pool())

	dbTemplate, err := queries.CreateTemplate(ctx, sqlc.CreateTemplateParams{
		Name:              template.Name,
		Subject:           template.Subject,
		HtmlBody:          template.HTMLBody,
		TextBody:          template.TextBody,
		BaseTemplateName:  template.BaseTemplateName,
		RequiredVariables: template.RequiredVariables,
		Version:           template.Version,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Update the template with generated timestamps
	template.CreatedAt = dbTemplate.CreatedAt
	template.UpdatedAt = dbTemplate.UpdatedAt

	return template, nil
}

// convertTemplateToDomain converts a sqlc Template to a domain Template
func convertTemplateToDomain(dbTemplate sqlc.EmailTemplate) *email.Template {
	return &email.Template{
		Name:              dbTemplate.Name,
		Subject:           dbTemplate.Subject,
		HTMLBody:          dbTemplate.HtmlBody,
		TextBody:          dbTemplate.TextBody,
		BaseTemplateName:  dbTemplate.BaseTemplateName,
		RequiredVariables: dbTemplate.RequiredVariables,
		Version:           dbTemplate.Version,
		CreatedAt:         dbTemplate.CreatedAt,
		UpdatedAt:         dbTemplate.UpdatedAt,
	}
}
