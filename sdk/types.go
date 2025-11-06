package sdk

import (
	"fmt"
	"time"
)

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	TemplateID  string            `json:"template_id"`
	To          string            `json:"to"`
	Variables   map[string]string `json:"variables,omitempty"`
	Priority    int32             `json:"priority,omitempty"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Validate validates the send email request
func (r *SendEmailRequest) Validate() error {
	if r.TemplateID == "" {
		return fmt.Errorf("template_id is required")
	}
	if r.To == "" {
		return fmt.Errorf("to is required")
	}
	// TODO: Add email format validation
	return nil
}

// SendEmailResponse represents the response from sending an email
type SendEmailResponse struct {
	// Currently empty as per proto definition
	// Could add JobID in future for tracking
}

// SendEmailBatchRequest represents a batch email request
type SendEmailBatchRequest struct {
	Emails []SendEmailRequest `json:"emails"`
}

// Validate validates the batch email request
func (r *SendEmailBatchRequest) Validate() error {
	if len(r.Emails) == 0 {
		return fmt.Errorf("emails list cannot be empty")
	}
	for i, email := range r.Emails {
		if err := email.Validate(); err != nil {
			return fmt.Errorf("invalid email at index %d: %w", i, err)
		}
	}
	return nil
}

// SendEmailBatchResponse represents the batch email response
type SendEmailBatchResponse struct {
	Results []SendEmailResponse `json:"results"`
}

// GetEmailStatusRequest represents a request to get email status
type GetEmailStatusRequest struct {
	JobID string `json:"job_id"`
}

// Validate validates the get email status request
func (r *GetEmailStatusRequest) Validate() error {
	if r.JobID == "" {
		return fmt.Errorf("job_id is required")
	}
	return nil
}

// EmailStatus represents the status of an email job
type EmailStatus string

const (
	EmailStatusUnspecified EmailStatus = "UNSPECIFIED"
	EmailStatusQueued      EmailStatus = "QUEUED"
	EmailStatusSending     EmailStatus = "SENDING"
	EmailStatusSent        EmailStatus = "SENT"
	EmailStatusFailed      EmailStatus = "FAILED"
	EmailStatusScheduled   EmailStatus = "SCHEDULED"
)

// GetEmailStatusResponse represents the response from getting email status
type GetEmailStatusResponse struct {
	JobID     string      `json:"job_id"`
	Status    EmailStatus `json:"status"`
	Attempts  int32       `json:"attempts"`
	LastError string      `json:"last_error,omitempty"`
	CreatedAt *time.Time  `json:"created_at,omitempty"`
	SentAt    *time.Time  `json:"sent_at,omitempty"`
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID                string   `json:"id"`
	Subject           string   `json:"subject"`
	RequiredVariables []string `json:"required_variables"`
	Version           int32    `json:"version"`
}

// ListTemplatesResponse represents the response from listing templates
type ListTemplatesResponse struct {
	Templates []EmailTemplate `json:"templates"`
}
