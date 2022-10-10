package main

import (
	"context"
	"fmt"

	helloserver "github.com/vcase/voice-hello/generated/grpc/hello"
)

type helloServer struct {
	helloserver.UnimplementedHelloServer
}

func (server *helloServer) Greet(ctx context.Context, request *helloserver.HelloRequest) (*helloserver.HelloResponse, error) {
	greeting := request.Greeting
	if greeting == "" {
		greeting = "hello"
	}

	name := request.Name
	if name == "" {
		name = "hello"
	}

	resp := helloserver.HelloResponse{
		Reply: fmt.Sprintf("%s, %s", greeting, name),
	}
	return &resp, nil
}
