package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendEmailRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailRequest{
			TemplateID: "welcome",
			To:         "user@example.com",
		}
		require.NoError(t, r.Validate())
	})

	t.Run("missing template ID", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailRequest{
			TemplateID: "",
			To:         "user@example.com",
		}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template_id")
	})

	t.Run("missing recipient", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailRequest{
			TemplateID: "welcome",
			To:         "",
		}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "to is required")
	})

	t.Run("invalid email format", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailRequest{
			TemplateID: "welcome",
			To:         "notanemail",
		}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email address")
	})

	t.Run("valid email address", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailRequest{
			TemplateID: "welcome",
			To:         "user@example.com",
		}
		require.NoError(t, r.Validate())
	})

	t.Run("valid email with special chars", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailRequest{
			TemplateID: "welcome",
			To:         "user+tag@example.com",
		}
		require.NoError(t, r.Validate())
	})
}

func TestSendEmailBatchRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid batch", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailBatchRequest{
			Emails: []SendEmailRequest{
				{TemplateID: "welcome", To: "alice@example.com"},
				{TemplateID: "welcome", To: "bob@example.com"},
			},
		}
		require.NoError(t, r.Validate())
	})

	t.Run("empty batch", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailBatchRequest{
			Emails: []SendEmailRequest{},
		}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "emails list cannot be empty")
	})

	t.Run("batch with invalid email at index 1", func(t *testing.T) {
		t.Parallel()
		r := &SendEmailBatchRequest{
			Emails: []SendEmailRequest{
				{TemplateID: "welcome", To: "alice@example.com"},
				{TemplateID: "welcome", To: "notanemail"},
			},
		}
		err := r.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index 1")
	})
}
