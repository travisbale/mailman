package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/pb"
	"github.com/travisbale/mailman/internal/queue/river"
	"google.golang.org/grpc"
)

type templatesDB interface {
	List(ctx context.Context) ([]*email.Template, error)
}

// Server implements the MailmanService gRPC service
type Server struct {
	pb.UnimplementedMailmanServiceServer
	jobQueue    *river.JobQueue
	templatesDB templatesDB
	grpcServer  *grpc.Server
	address     string
}

// NewServer creates a new gRPC server
func NewServer(address string, queueClient *river.JobQueue, templatesDB templatesDB) *Server {
	grpcServer := grpc.NewServer()

	server := &Server{
		jobQueue:    queueClient,
		templatesDB: templatesDB,
		grpcServer:  grpcServer,
		address:     address,
	}

	pb.RegisterMailmanServiceServer(grpcServer, server)

	return server
}

// ListenAndServe starts the gRPC server
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	fmt.Printf("Starting gRPC server on %s\n", s.address)
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}
