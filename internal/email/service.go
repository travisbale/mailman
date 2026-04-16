package email

import (
	"context"
	"fmt"
)

// Renderer defines the interface for rendering email templates.
type Renderer interface {
	Render(ctx context.Context, templateName string, variables map[string]string) (*RenderedTemplate, error)
}

type jobQueue interface {
	EnqueueEmailJob(ctx context.Context, jobArgs *JobArgs) error
}

// Service orchestrates template validation, rendering, and job enqueueing.
type Service struct {
	Templates   templateDB
	Renderer    Renderer
	Queue       jobQueue
	FromAddress string
	FromName    string
}

// Send validates the template, renders it, and enqueues the pre-rendered email.
func (s *Service) Send(ctx context.Context, req SendRequest) error {
	tmpl, err := s.Templates.GetTemplate(ctx, req.TemplateName)
	if err != nil {
		return fmt.Errorf("template %q: %w", req.TemplateName, err)
	}

	// Validate required variables before rendering
	for _, required := range tmpl.RequiredVariables {
		if _, ok := req.Variables[required]; !ok {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}

	rendered, err := s.Renderer.Render(ctx, req.TemplateName, req.Variables)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	jobArgs := &JobArgs{
		To:          req.To,
		From:        s.FromAddress,
		FromName:    s.FromName,
		Subject:     rendered.Subject,
		HTMLBody:    rendered.HTMLBody,
		TextBody:    rendered.TextBody,
		Priority:    req.Priority,
		ScheduledAt: req.ScheduledAt,
	}

	if err := s.Queue.EnqueueEmailJob(ctx, jobArgs); err != nil {
		return fmt.Errorf("failed to enqueue email: %w", err)
	}

	return nil
}
