package email

import "errors"

var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrMissingVariable  = errors.New("missing variable")
)
