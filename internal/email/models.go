package email

import "time"

// Email represents an email to be sent with template information
type Email struct {
	To           string
	From         string
	FromName     string
	TemplateName string
	Variables    map[string]string
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
type JobArgs struct {
	To           string
	TemplateName string
	Variables    map[string]string
	Priority     int32
	Metadata     map[string]string
	ScheduledAt  *time.Time
}

// Kind returns the unique identifier for this job type
func (JobArgs) Kind() string { return "send_email" }

// RenderedTemplate contains the rendered email content
type RenderedTemplate struct {
	Subject  string
	HTMLBody string
	TextBody string
}
