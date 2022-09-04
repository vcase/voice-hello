package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	helloserver "github.com/vcase/voice-hello/generated/hello"
	"google.golang.org/grpc"
)

_

func runGrpc() error {

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		return err
	}

	server := &helloServer{}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	helloserver.RegisterHelloServer(grpcServer, server)

	err = grpcServer.Serve(lis)

	return err
}
