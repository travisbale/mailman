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
	var template *email.Template

	err := r.db.WithTransaction(ctx, func(q *sqlc.Queries) error {
		dbTemplate, err := q.GetTemplate(ctx, name)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf("template not found: %s", name)
			}
			return fmt.Errorf("failed to get template: %w", err)
		}

		template = convertTemplateToDomain(dbTemplate)
		return nil
	})

	return template, err
}

// List retrieves all email templates, optionally filtered by name
func (r *TemplatesDB) List(ctx context.Context) ([]*email.Template, error) {
	var templates []*email.Template

	err := r.db.WithTransaction(ctx, func(q *sqlc.Queries) error {
		dbTemplates, err := q.ListTemplates(ctx)
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		templates = make([]*email.Template, len(dbTemplates))
		for i := range dbTemplates {
			templates[i] = convertTemplateToDomain(dbTemplates[i])
		}

		return nil
	})

	return templates, err
}

// Create inserts a new email template
func (r *TemplatesDB) Create(ctx context.Context, template *email.Template) (*email.Template, error) {
	err := r.db.WithTransaction(ctx, func(q *sqlc.Queries) error {
		dbTemplate, err := q.CreateTemplate(ctx, sqlc.CreateTemplateParams{
			Name:              template.Name,
			Subject:           template.Subject,
			HtmlBody:          template.HTMLBody,
			TextBody:          template.TextBody,
			BaseTemplateName:  template.BaseTemplateName,
			RequiredVariables: template.RequiredVariables,
			Version:           template.Version,
		})

		if err != nil {
			return fmt.Errorf("failed to create template: %w", err)
		}

		template.CreatedAt = dbTemplate.CreatedAt
		template.UpdatedAt = dbTemplate.UpdatedAt

		return nil
	})

	return template, err
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
