package grpc

import (
	"context"

	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type jobQueue interface {
	EnqueueEmailJob(ctx context.Context, jobArgs *email.JobArgs) error
}

type EmailHandler struct {
	pb.UnimplementedMailmanServiceServer
	jobQueue jobQueue
}

func NewEmailHandler(queue jobQueue) *EmailHandler {
	return &EmailHandler{
		jobQueue: queue,
	}
}

// SendEmail enqueues a single email for delivery
func (h *EmailHandler) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	if err := h.validateSendEmailRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	params := &email.JobArgs{
		To:           req.To,
		TemplateName: req.TemplateId,
		Variables:    req.Variables,
		Priority:     req.Priority,
		Metadata:     req.Metadata,
	}

	if req.ScheduledAt != nil {
		scheduledAt := req.ScheduledAt.AsTime()
		params.ScheduledAt = &scheduledAt
	}

	err := h.jobQueue.EnqueueEmailJob(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue email: %v", err)
	}

	return &pb.SendEmailResponse{}, nil
}

// SendEmailBatch enqueues multiple emails in a single request
func (s *Server) SendEmailBatch(ctx context.Context, req *pb.SendEmailBatchRequest) (*pb.SendEmailBatchResponse, error) {
	results := make([]*pb.SendEmailResponse, 0, len(req.Emails))

	for _, emailReq := range req.Emails {
		resp, err := s.emailHandler.SendEmail(ctx, emailReq)
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
