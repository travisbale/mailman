package email

import "time"

// Email represents an email to be sent
type Email struct {
	From     string
	FromName string
	To       string
	Subject  string
	HTMLBody string
	TextBody string
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

// JobArgs holds the job arguments required to send an email
// Contains pre-rendered content to avoid re-rendering in the worker
type JobArgs struct {
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	Priority    int32
	Metadata    map[string]string
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
