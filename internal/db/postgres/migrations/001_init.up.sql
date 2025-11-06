-- Email templates table
-- Stores reusable email templates with versioning support
CREATE TABLE email_templates (
    name TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    html_body TEXT NOT NULL,
    text_body TEXT,
    base_template_name TEXT REFERENCES email_templates(name),
    required_variables TEXT[] DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
