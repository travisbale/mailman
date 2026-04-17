package html_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	htmlrenderer "github.com/travisbale/mailman/internal/renderers/html"

	"github.com/travisbale/mailman/internal/email"
)

type mockTemplateDB struct {
	templates map[string]*email.Template
}

func (m *mockTemplateDB) GetTemplate(ctx context.Context, name string) (*email.Template, error) {
	tmpl, ok := m.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

func strPtr(s string) *string { return &s }

func TestRenderer_SimpleTemplate(t *testing.T) {
	t.Parallel()

	db := &mockTemplateDB{
		templates: map[string]*email.Template{
			"welcome": {
				Name:      "welcome",
				Subject:   "Welcome {{index . \"UserName\"}}!",
				HTMLBody:  "<h1>Hello {{index . \"UserName\"}}</h1>",
				Variables: []string{"UserName"},
			},
		},
	}

	r := htmlrenderer.New(db)
	result, err := r.Render(context.Background(), "welcome", map[string]string{
		"UserName": "Alice",
	})

	require.NoError(t, err)
	assert.Equal(t, "Welcome Alice!", result.Subject)
	assert.Equal(t, "<h1>Hello Alice</h1>", result.HTMLBody)
	assert.Equal(t, "", result.TextBody)
}

func TestRenderer_TemplateWithTextBody(t *testing.T) {
	t.Parallel()

	db := &mockTemplateDB{
		templates: map[string]*email.Template{
			"notify": {
				Name:      "notify",
				Subject:   "Notification",
				HTMLBody:  "<p>{{index . \"Message\"}}</p>",
				TextBody:  strPtr("{{index . \"Message\"}}"),
				Variables: []string{"Message"},
			},
		},
	}

	r := htmlrenderer.New(db)
	result, err := r.Render(context.Background(), "notify", map[string]string{
		"Message": "Hello world",
	})

	require.NoError(t, err)
	assert.Equal(t, "Notification", result.Subject)
	assert.Equal(t, "<p>Hello world</p>", result.HTMLBody)
	assert.Equal(t, "Hello world", result.TextBody)
}

func TestRenderer_TemplateWithoutTextBody(t *testing.T) {
	t.Parallel()

	db := &mockTemplateDB{
		templates: map[string]*email.Template{
			"html_only": {
				Name:     "html_only",
				Subject:  "HTML Only",
				HTMLBody: "<p>Content</p>",
				TextBody: nil,
			},
		},
	}

	r := htmlrenderer.New(db)
	result, err := r.Render(context.Background(), "html_only", map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, "<p>Content</p>", result.HTMLBody)
	assert.Equal(t, "", result.TextBody)
}

func TestRenderer_NestedTemplateWithBase(t *testing.T) {
	t.Parallel()

	db := &mockTemplateDB{
		templates: map[string]*email.Template{
			"base_layout": {
				Name:    "base_layout",
				Subject: "",
				HTMLBody: `<html><body>` +
					`<header>MyApp</header>` +
					`{{template "content" .}}` +
					`<footer>Footer</footer>` +
					`</body></html>`,
			},
			"child_email": {
				Name:             "child_email",
				Subject:          "Hello {{index . \"UserName\"}}",
				HTMLBody:         `{{define "content"}}<p>Welcome {{index . "UserName"}}!</p>{{end}}`,
				BaseTemplateName: strPtr("base_layout"),
				Variables:        []string{"UserName"},
			},
		},
	}

	r := htmlrenderer.New(db)
	result, err := r.Render(context.Background(), "child_email", map[string]string{
		"UserName": "Bob",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello Bob", result.Subject)
	assert.Contains(t, result.HTMLBody, "<header>MyApp</header>")
	assert.Contains(t, result.HTMLBody, "<p>Welcome Bob!</p>")
	assert.Contains(t, result.HTMLBody, "<footer>Footer</footer>")
}

func TestRenderer_MissingVariable(t *testing.T) {
	t.Parallel()

	db := &mockTemplateDB{
		templates: map[string]*email.Template{
			"requires_vars": {
				Name:      "requires_vars",
				Subject:   "Hi {{index . \"UserName\"}}",
				HTMLBody:  "<p>body</p>",
				Variables: []string{"UserName", "ResetLink"},
			},
		},
	}

	r := htmlrenderer.New(db)
	_, err := r.Render(context.Background(), "requires_vars", map[string]string{
		// UserName is missing
		"ResetLink": "https://example.com/reset",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "UserName")
}

func TestRenderer_TemplateNotFound(t *testing.T) {
	t.Parallel()

	db := &mockTemplateDB{
		templates: map[string]*email.Template{},
	}

	r := htmlrenderer.New(db)
	_, err := r.Render(context.Background(), "nonexistent", map[string]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRenderer_CircularReferenceDetection(t *testing.T) {
	t.Parallel()

	// Template A -> Template B -> Template A (cycle)
	db := &mockTemplateDB{
		templates: map[string]*email.Template{
			"template_a": {
				Name:             "template_a",
				Subject:          "Subject",
				HTMLBody:         `{{define "content"}}Content A{{end}}`,
				BaseTemplateName: strPtr("template_b"),
			},
			"template_b": {
				Name:             "template_b",
				Subject:          "",
				HTMLBody:         `{{template "content" .}}`,
				BaseTemplateName: strPtr("template_a"),
			},
		},
	}

	r := htmlrenderer.New(db)
	_, err := r.Render(context.Background(), "template_a", map[string]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}
