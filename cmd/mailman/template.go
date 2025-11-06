package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/travisbale/mailman/internal/db/postgres"
	"github.com/travisbale/mailman/internal/email"
	"github.com/urfave/cli/v2"
)

// templateCmd provides template management commands
var templateCmd = &cli.Command{
	Name:  "template",
	Usage: "Manage email templates",
	Subcommands: []*cli.Command{
		templateAddCmd,
		templateListCmd,
	},
}

// templateAddCmd adds a new email template
var templateAddCmd = &cli.Command{
	Name:  "add",
	Usage: "Add a new email template",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "database-url",
			Usage:   "PostgreSQL connection string",
			EnvVars: []string{"DATABASE_URL"},
			Value:   "postgres://postgres:secure_password@postgres:5432/mailman?sslmode=disable",
		},
		&cli.StringFlag{
			Name:     "name",
			Usage:    "Template name (e.g., welcome_email, password_reset)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "subject",
			Usage:    "Email subject line (supports Go template syntax)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "html-file",
			Usage:    "Path to HTML body file",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "text-file",
			Usage: "Path to plain text body file (optional)",
		},
		&cli.StringFlag{
			Name:  "base",
			Usage: "Base template name to inherit from (optional, for nested templates)",
		},
		&cli.StringFlag{
			Name:  "vars",
			Usage: "Comma-separated list of required template variables (e.g., UserName,AppName)",
		},
		&cli.IntFlag{
			Name:  "version",
			Usage: "Template version number",
			Value: 1,
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		// Connect to database
		db, err := postgres.NewDB(ctx, c.String("database-url"))
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create template service
		templatesDB := postgres.NewTemplatesDB(db)
		templateService := email.NewTemplateService(templatesDB)

		// Validate and create template
		template, err := buildTemplate(c)
		if err != nil {
			return fmt.Errorf("failed to build template: %w", err)
		}

		created, err := templateService.CreateTemplate(ctx, template)
		if err != nil {
			return fmt.Errorf("failed to create template: %w", err)
		}

		fmt.Printf("Template created successfully\n")
		fmt.Printf("  Name: %s\n", created.Name)
		if created.BaseTemplateName != nil {
			fmt.Printf("  Base template: %s\n", *created.BaseTemplateName)
		}
		fmt.Printf("  Version: %d\n", created.Version)
		if len(created.RequiredVariables) > 0 {
			fmt.Printf("  Required variables: %s\n", strings.Join(created.RequiredVariables, ", "))
		}

		return nil
	},
}

func buildTemplate(c *cli.Context) (*email.Template, error) {
	// Read HTML body from file
	htmlFile := c.String("html-file")
	htmlContent, err := os.ReadFile(htmlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML file: %w", err)
	}
	htmlBody := string(htmlContent)

	// Read text body from file (optional)
	var textBody string
	textFile := c.String("text-file")
	if textFile != "" {
		textContent, err := os.ReadFile(textFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read text file: %w", err)
		}
		textBody = string(textContent)
	}

	// Parse required variables
	var requiredVars []string
	if varsStr := c.String("vars"); varsStr != "" {
		for v := range strings.SplitSeq(varsStr, ",") {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				requiredVars = append(requiredVars, trimmed)
			}
		}
	}

	// Prepare optional fields
	var textBodyPtr *string
	if textBody != "" {
		textBodyPtr = &textBody
	}

	var baseTemplatePtr *string
	if baseName := c.String("base"); baseName != "" {
		baseTemplatePtr = &baseName
	}

	return &email.Template{
		Name:              c.String("name"),
		Subject:           c.String("subject"),
		HTMLBody:          htmlBody,
		TextBody:          textBodyPtr,
		BaseTemplateName:  baseTemplatePtr,
		RequiredVariables: requiredVars,
		Version:           int32(c.Int("version")),
	}, nil
}

// templateListCmd lists all email templates
var templateListCmd = &cli.Command{
	Name:  "list",
	Usage: "List all email templates",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "database-url",
			Usage:   "PostgreSQL connection string",
			EnvVars: []string{"DATABASE_URL"},
			Value:   "postgres://postgres:secure_password@postgres:5432/mailman?sslmode=disable",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := context.Background()

		// Connect to database
		db, err := postgres.NewDB(ctx, c.String("database-url"))
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create template service
		templatesDB := postgres.NewTemplatesDB(db)
		templateService := email.NewTemplateService(templatesDB)

		// List templates
		templates, err := templateService.ListTemplates(ctx)
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return nil
		}

		// Print templates in table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "NAME\tBASE\tVERSION\tVARIABLES\tCREATED"); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
		if _, err := fmt.Fprintln(w, "----\t----\t-------\t---------\t-------"); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}

		for _, tmpl := range templates {
			base := "-"
			if tmpl.BaseTemplateName != nil {
				base = *tmpl.BaseTemplateName
			}

			vars := "-"
			if len(tmpl.RequiredVariables) > 0 {
				vars = strings.Join(tmpl.RequiredVariables, ", ")
			}

			if _, err := fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				tmpl.Name,
				base,
				tmpl.Version,
				vars,
				tmpl.CreatedAt.Format("2006-01-02"),
			); err != nil {
				return fmt.Errorf("failed to write template row: %w", err)
			}
		}

		if err := w.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}

		return nil
	},
}
