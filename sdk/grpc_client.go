package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCClient is a gRPC client for the mailman API
type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.MailmanServiceClient
}

// GRPCClientOption is a functional option for configuring the gRPC client
type GRPCClientOption func(*grpcClientConfig)

type grpcClientConfig struct {
	dialOptions []grpc.DialOption
	timeout     time.Duration
}

// WithDialOptions allows setting custom gRPC dial options
func WithDialOptions(opts ...grpc.DialOption) GRPCClientOption {
	return func(c *grpcClientConfig) {
		c.dialOptions = append(c.dialOptions, opts...)
	}
}

// WithTimeout sets the default timeout for gRPC calls
func WithTimeout(timeout time.Duration) GRPCClientOption {
	return func(c *grpcClientConfig) {
		c.timeout = timeout
	}
}

// NewGRPCClient creates a new gRPC client for the mailman API
// address should be in the format "host:port" (e.g., "localhost:50051")
func NewGRPCClient(address string, opts ...GRPCClientOption) (*GRPCClient, error) {
	config := &grpcClientConfig{
		timeout:     30 * time.Second,
		dialOptions: []grpc.DialOption{},
	}

	for _, opt := range opts {
		opt(config)
	}

	// Add default dial options if none provided
	if len(config.dialOptions) == 0 {
		config.dialOptions = append(config.dialOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	// Establish connection
	conn, err := grpc.NewClient(address, config.dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	return &GRPCClient{
		conn:   conn,
		client: pb.NewMailmanServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendEmail sends a single email
func (c *GRPCClient) SendEmail(ctx context.Context, req SendEmailRequest) (*SendEmailResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to protobuf request
	pbReq := &pb.SendEmailRequest{
		TemplateId: req.TemplateID,
		To:         req.To,
		Variables:  req.Variables,
		Priority:   req.Priority,
		Metadata:   req.Metadata,
	}

	if req.ScheduledAt != nil {
		pbReq.ScheduledAt = timestamppb.New(*req.ScheduledAt)
	}

	// Call gRPC service
	_, err := c.client.SendEmail(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	return &SendEmailResponse{}, nil
}

// SendEmailBatch sends multiple emails in a single request
func (c *GRPCClient) SendEmailBatch(ctx context.Context, req SendEmailBatchRequest) (*SendEmailBatchResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to protobuf requests
	pbEmails := make([]*pb.SendEmailRequest, len(req.Emails))
	for i, email := range req.Emails {
		pbEmails[i] = &pb.SendEmailRequest{
			TemplateId: email.TemplateID,
			To:         email.To,
			Variables:  email.Variables,
			Priority:   email.Priority,
			Metadata:   email.Metadata,
		}
		if email.ScheduledAt != nil {
			pbEmails[i].ScheduledAt = timestamppb.New(*email.ScheduledAt)
		}
	}

	pbReq := &pb.SendEmailBatchRequest{
		Emails: pbEmails,
	}

	// Call gRPC service
	pbResp, err := c.client.SendEmailBatch(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send email batch: %w", err)
	}

	// Convert response
	results := make([]SendEmailResponse, len(pbResp.Results))
	return &SendEmailBatchResponse{
		Results: results,
	}, nil
}

// GetEmailStatus retrieves the status of an email job
func (c *GRPCClient) GetEmailStatus(ctx context.Context, req GetEmailStatusRequest) (*GetEmailStatusResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to protobuf request
	pbReq := &pb.GetEmailStatusRequest{
		JobId: req.JobID,
	}

	// Call gRPC service
	pbResp, err := c.client.GetEmailStatus(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get email status: %w", err)
	}

	// Convert response
	resp := &GetEmailStatusResponse{
		JobID:     pbResp.JobId,
		Status:    convertEmailStatus(pbResp.Status),
		Attempts:  pbResp.Attempts,
		LastError: pbResp.LastError,
	}

	if pbResp.CreatedAt != nil {
		createdAt := pbResp.CreatedAt.AsTime()
		resp.CreatedAt = &createdAt
	}

	if pbResp.SentAt != nil {
		sentAt := pbResp.SentAt.AsTime()
		resp.SentAt = &sentAt
	}

	return resp, nil
}

// ListTemplates returns all available email templates
func (c *GRPCClient) ListTemplates(ctx context.Context) (*ListTemplatesResponse, error) {
	// Call gRPC service
	pbResp, err := c.client.ListTemplates(ctx, &pb.ListTemplatesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// Convert response
	templates := make([]EmailTemplate, len(pbResp.Templates))
	for i, tmpl := range pbResp.Templates {
		templates[i] = EmailTemplate{
			ID:                tmpl.Id,
			Subject:           tmpl.Subject,
			RequiredVariables: tmpl.RequiredVariables,
			Version:           tmpl.Version,
		}
	}

	return &ListTemplatesResponse{
		Templates: templates,
	}, nil
}

// convertEmailStatus converts protobuf EmailStatus to SDK EmailStatus
func convertEmailStatus(pbStatus pb.EmailStatus) EmailStatus {
	switch pbStatus {
	case pb.EmailStatus_EMAIL_STATUS_QUEUED:
		return EmailStatusQueued
	case pb.EmailStatus_EMAIL_STATUS_SENDING:
		return EmailStatusSending
	case pb.EmailStatus_EMAIL_STATUS_SENT:
		return EmailStatusSent
	case pb.EmailStatus_EMAIL_STATUS_FAILED:
		return EmailStatusFailed
	case pb.EmailStatus_EMAIL_STATUS_SCHEDULED:
		return EmailStatusScheduled
	default:
		return EmailStatusUnspecified
	}
}
