package email_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/travisbale/mailman/internal/email"
)

// mockTemplateDB returns a fixed template or error from GetTemplate.
type mockTemplateDB struct {
	template *email.Template
	err      error
}

func (m *mockTemplateDB) GetTemplate(_ context.Context, _ string) (*email.Template, error) {
	return m.template, m.err
}

func (m *mockTemplateDB) Create(_ context.Context, _ *email.Template) (*email.Template, error) {
	panic("not implemented")
}

func (m *mockTemplateDB) List(_ context.Context) ([]*email.Template, error) {
	panic("not implemented")
}

// mockRenderer returns a fixed rendered template or error from Render.
type mockRenderer struct {
	rendered *email.RenderedTemplate
	err      error
}

func (m *mockRenderer) Render(_ context.Context, _ string, _ map[string]string) (*email.RenderedTemplate, error) {
	return m.rendered, m.err
}

// mockQueue captures the JobArgs passed to EnqueueEmailJob.
type mockQueue struct {
	jobArgs *email.JobArgs
	err     error
}

func (m *mockQueue) EnqueueEmailJob(_ context.Context, jobArgs *email.JobArgs) error {
	m.jobArgs = jobArgs
	return m.err
}

func TestService_Send_Success(t *testing.T) {
	t.Parallel()

	scheduledAt := time.Now().Add(5 * time.Minute)
	queue := &mockQueue{}

	svc := &email.Service{
		Templates: &mockTemplateDB{
			template: &email.Template{
				Name:      "welcome",
				Variables: []string{"Name"},
			},
		},
		Renderer: &mockRenderer{
			rendered: &email.RenderedTemplate{
				Subject:  "Hello, World!",
				HTMLBody: "<p>Hello</p>",
				TextBody: "Hello",
			},
		},
		Queue:       queue,
		FromAddress: "no-reply@example.com",
		FromName:    "Example",
	}

	req := email.SendRequest{
		To:           "user@example.com",
		TemplateName: "welcome",
		Variables:    map[string]string{"Name": "Alice"},
		Priority:     2,
		ScheduledAt:  &scheduledAt,
	}

	err := svc.Send(context.Background(), req)
	require.NoError(t, err)

	// Verify the enqueued job contains pre-rendered content and service config.
	require.NotNil(t, queue.jobArgs)
	assert.Equal(t, "user@example.com", queue.jobArgs.To)
	assert.Equal(t, "no-reply@example.com", queue.jobArgs.From)
	assert.Equal(t, "Example", queue.jobArgs.FromName)
	assert.Equal(t, "Hello, World!", queue.jobArgs.Subject)
	assert.Equal(t, "<p>Hello</p>", queue.jobArgs.HTMLBody)
	assert.Equal(t, "Hello", queue.jobArgs.TextBody)
	assert.Equal(t, int32(2), queue.jobArgs.Priority)
	assert.Equal(t, &scheduledAt, queue.jobArgs.ScheduledAt)
}

func TestService_Send_MissingVariable(t *testing.T) {
	t.Parallel()

	svc := &email.Service{
		Templates: &mockTemplateDB{
			template: &email.Template{
				Name:      "invite",
				Variables: []string{"Name", "Company"},
			},
		},
		Renderer: &mockRenderer{},
		Queue:    &mockQueue{},
	}

	// Only "Name" provided; "Company" is missing.
	req := email.SendRequest{
		TemplateName: "invite",
		Variables:    map[string]string{"Name": "Alice"},
	}

	err := svc.Send(context.Background(), req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, email.ErrMissingVariable))
	assert.Contains(t, err.Error(), "Company")
}
