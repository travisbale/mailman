package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	resp, err := testClient.ListTemplates(ctx)
	require.NoError(t, err)
	assert.Len(t, resp.Templates, 4)

	// Build a map for easier assertion
	templates := make(map[string]struct {
		subject string
		vars    []string
		version int32
	})
	for _, tmpl := range resp.Templates {
		templates[tmpl.ID] = struct {
			subject string
			vars    []string
			version int32
		}{
			subject: tmpl.Subject,
			vars:    tmpl.Variables,
			version: tmpl.Version,
		}
	}

	simple := templates["simple_template"]
	assert.Equal(t, "Hello {{.Name}}!", simple.subject)
	assert.Equal(t, []string{"Name"}, simple.vars)
	assert.Equal(t, int32(1), simple.version)

	multi := templates["multi_var_template"]
	assert.Equal(t, "Welcome {{.Name}} from {{.Company}}!", multi.subject)
	assert.ElementsMatch(t, []string{"Name", "Company"}, multi.vars)
}
