package grpc

import (
	"context"

	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SendEmail validates the request, then delegates to the email service for
// template rendering and job enqueueing.
func (s *Server) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	if err := s.validateSendEmailRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	sendReq := email.SendRequest{
		To:           req.To,
		TemplateName: req.TemplateId,
		Variables:    req.Variables,
		Priority:     req.Priority,
	}

	if req.ScheduledAt != nil {
		scheduledAt := req.ScheduledAt.AsTime()
		sendReq.ScheduledAt = &scheduledAt
	}

	if err := s.emailService.Send(ctx, sendReq); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send email: %v", err)
	}

	return &pb.SendEmailResponse{}, nil
}

// SendEmailBatch enqueues multiple emails in a single request
func (s *Server) SendEmailBatch(ctx context.Context, req *pb.SendEmailBatchRequest) (*pb.SendEmailBatchResponse, error) {
	results := make([]*pb.SendEmailResponse, 0, len(req.Emails))

	for _, emailReq := range req.Emails {
		resp, err := s.SendEmail(ctx, emailReq)
		if err != nil {
			return nil, err
		}
		results = append(results, resp)
	}

	return &pb.SendEmailBatchResponse{
		Results: results,
	}, nil
}

// ListTemplates returns all available email templates
func (s *Server) ListTemplates(ctx context.Context, req *pb.ListTemplatesRequest) (*pb.ListTemplatesResponse, error) {
	templates, err := s.templatesDB.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list templates: %v", err)
	}

	pbTemplates := make([]*pb.EmailTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, &pb.EmailTemplate{
			Id:                t.Name,
			Subject:           t.Subject,
			RequiredVariables: t.RequiredVariables,
			Version:           t.Version,
		})
	}

	return &pb.ListTemplatesResponse{
		Templates: pbTemplates,
	}, nil
}
