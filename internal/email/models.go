package email

import "time"

// SendRequest represents a request to send an email, before rendering.
type SendRequest struct {
	To           string
	TemplateName string
	Variables    map[string]string
	Priority     int32
	ScheduledAt  *time.Time
}

// Template represents an email template stored in the database
type Template struct {
	Name              string
	Subject           string
	HTMLBody          string
	TextBody          *string
	BaseTemplateName  *string
	RequiredVariables []string
	Version           int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// JobArgs holds pre-rendered email content for the job queue.
type JobArgs struct {
	To          string
	From        string
	FromName    string
	Subject     string
	HTMLBody    string
	TextBody    string
	Priority    int32
	ScheduledAt *time.Time
}

// Kind returns the unique identifier for this job type
func (JobArgs) Kind() string { return "send_email" }

// RenderedTemplate contains the rendered email content
type RenderedTemplate struct {
	Subject  string
	HTMLBody string
	TextBody string
}
