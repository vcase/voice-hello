package main

import (
	"context"
	"log"
	"net"

	helloserver "github.com/vcase/voice-hello/generated/hello"
	"google.golang.org/grpc"
)

type helloServer struct {
	helloserver.UnimplementedHelloServer
}

func main() {

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal("Failed to listen: %v", err)
	}

	server := &helloServer{}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	helloserver.RegisterHelloServer(grpcServer, server)

	grpcServer.Serve(lis)
}

func (server *helloServer) Hello(context.Context, *helloserver.HelloRequest) (*helloserver.HelloResponse, error) {
	resp := helloserver.HelloResponse{
		Reply: "Hello, there",
	}
	return &resp, nil
}
