package server

import (
	"log"
	"net"

	pb "github.com/prorobot-ai/grpc-protos/gen/crawler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StartGRPCServer initializes and runs the gRPC server
func StartGRPCServer() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterCrawlerServiceServer(server, NewCrawlerServer())

	reflection.Register(server)
	log.Println("🚀 gRPC server running on port 50051")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
