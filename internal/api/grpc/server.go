package grpc

import (
	"fmt"
	"net"

	"github.com/travisbale/mailman/internal/db/postgres"
	"github.com/travisbale/mailman/internal/pb"
	"github.com/travisbale/mailman/internal/queue/river"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server implements the MailmanService gRPC service
type Server struct {
	emailHandler *EmailHandler
	templatesDB  *postgres.TemplatesDB
	grpcServer   *grpc.Server
	address      string
}

// NewServer creates a new gRPC server
func NewServer(address string, queueClient *river.JobQueue, templatesDB *postgres.TemplatesDB) *Server {
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	emailHandler := NewEmailHandler(queueClient)

	pb.RegisterMailmanServiceServer(grpcServer, emailHandler)

	return &Server{
		emailHandler: emailHandler,
		templatesDB:  templatesDB,
		grpcServer:   grpcServer,
		address:      address,
	}
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
