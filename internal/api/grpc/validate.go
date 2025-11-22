package grpc

import (
	"fmt"

	"github.com/travisbale/mailman/internal/pb"
)

// validateSendEmailRequest validates the send email request
func (s *EmailHandler) validateSendEmailRequest(req *pb.SendEmailRequest) error {
	if req.TemplateId == "" {
		return fmt.Errorf("template_id is required")
	}
	if req.To == "" {
		return fmt.Errorf("to is required")
	}

	// TODO: Add email format validation
	return nil
}
