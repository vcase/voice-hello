package main

import (
	"context"

	helloserver "github.com/vcase/voice-hello/generated/hello"
)

type helloServer struct {
	helloserver.UnimplementedHelloServer
}

func (server *helloServer) Hello(context.Context, *helloserver.HelloRequest) (*helloserver.HelloResponse, error) {
	resp := helloserver.HelloResponse{
		Reply: "Hello, there",
	}
	return &resp, nil
}
