package grpc

import (
	"net"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	grpc_interface "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/grpc"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/proto/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/gzip"
)

type Server struct {
	grpcServer *grpc.Server
	logger     ports.LoggerService
}

func NewServer(grpc_handler *grpc_interface.Handler, logger ports.LoggerService, opt ...grpc.ServerOption) *Server {
	grpcServer := grpc.NewServer(
		opt...,
	)
	grpc.EnableTracing = true // Enable tracing for debugging

	// Register gzip compressor
	encoding.RegisterCompressor(encoding.GetCompressor(gzip.Name))

	handler := grpc_handler
	proto.RegisterNotificationServiceServer(grpcServer, handler)

	return &Server{
		grpcServer: grpcServer,
		logger:     logger,
	}
}

func (s *Server) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		s.logger.Error("Failed to listen", logger.String("address", address), logger.Error(err))
		return err
	}
	s.logger.Info("gRPC server started", logger.String("address", address))
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
	s.logger.Info("gRPC server stopped")
}
