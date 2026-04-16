package grpc

import (
	"context"
	"errors"

	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/pb"
	"github.com/travisbale/mailman/sdk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SendEmail validates the request, then delegates to the email service for
// template rendering and job enqueueing.
func (s *Server) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	sdkReq := sdk.SendEmailRequest{
		TemplateID: req.TemplateId,
		To:         req.To,
	}

	if err := sdkReq.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
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
		switch {
		case errors.Is(err, email.ErrTemplateNotFound):
			return nil, status.Errorf(codes.NotFound, "template not found: %s", req.TemplateId)
		case errors.Is(err, email.ErrMissingVariable):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "failed to send email")
		}
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
		return nil, status.Errorf(codes.Internal, "failed to list templates")
	}

	pbTemplates := make([]*pb.EmailTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, &pb.EmailTemplate{
			Id:        t.Name,
			Subject:   t.Subject,
			Variables: t.Variables,
			Version:   t.Version,
		})
	}

	return &pb.ListTemplatesResponse{
		Templates: pbTemplates,
	}, nil
}
