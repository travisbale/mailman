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
		return err
	}

	for _, v := range tmpl.Variables {
		if _, ok := req.Variables[v]; !ok {
			return fmt.Errorf("%w: %s", ErrMissingVariable, v)
		}
	}

	rendered, err := s.Renderer.Render(ctx, req.TemplateName, req.Variables)
	if err != nil {
		return err
	}

	if err := s.Queue.EnqueueEmailJob(ctx, &JobArgs{
		To:          req.To,
		From:        s.FromAddress,
		FromName:    s.FromName,
		Subject:     rendered.Subject,
		HTMLBody:    rendered.HTMLBody,
		TextBody:    rendered.TextBody,
		Priority:    req.Priority,
		ScheduledAt: req.ScheduledAt,
	}); err != nil {
		return err
	}

	return nil
}
