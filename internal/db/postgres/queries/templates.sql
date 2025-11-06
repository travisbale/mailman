-- name: GetTemplate :one
SELECT name, subject, html_body, text_body, base_template_name, required_variables, version, created_at, updated_at
FROM email_templates
WHERE name = $1;

-- name: ListTemplates :many
SELECT name, subject, html_body, text_body, base_template_name, required_variables, version, created_at, updated_at
FROM email_templates
ORDER BY name, version DESC;

-- name: CreateTemplate :one
INSERT INTO email_templates (name, subject, html_body, text_body, base_template_name, required_variables, version)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING name, subject, html_body, text_body, base_template_name, required_variables, version, created_at, updated_at;
