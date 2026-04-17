package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/travisbale/mailman/sdk"
)

func TestSendEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailRequest{
		TemplateID: "simple_template",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.NoError(t, err)
}

func TestSendEmailWithMultipleVariables(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailRequest{
		TemplateID: "multi_var_template",
		To:         "user@example.com",
		Variables: map[string]string{
			"Name":    "Alice",
			"Company": "Acme Corp",
		},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.NoError(t, err)
}

func TestSendEmailWithNestedTemplate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailRequest{
		TemplateID: "nested_template",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.NoError(t, err)
}

func TestSendEmailNonexistentTemplate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailRequest{
		TemplateID: "does_not_exist",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does_not_exist")
}

func TestSendEmailMissingVariables(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// multi_var_template requires both Name and Company
	req := sdk.SendEmailRequest{
		TemplateID: "multi_var_template",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Company")
}

func TestSendEmailBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailBatchRequest{
		Emails: []sdk.SendEmailRequest{
			{
				TemplateID: "simple_template",
				To:         "user1@example.com",
				Variables:  map[string]string{"Name": "Alice"},
			},
			{
				TemplateID: "simple_template",
				To:         "user2@example.com",
				Variables:  map[string]string{"Name": "Bob"},
			},
		},
	}

	resp, err := testClient.SendEmailBatch(ctx, req)
	require.NoError(t, err)
	assert.Len(t, resp.Results, 2)
}
