package main

import (
	"context"

	binaryopsserver "github.com/vcase/voice-servicetemplate/generated/grpc/service-template/binaryops"
)

type additionServer struct {
	binaryopsserver.UnimplementedAdditionServer
}

func (server *additionServer) Add(ctx context.Context, request *binaryopsserver.BinaryOpRequest) (*binaryopsserver.BinaryOpResponse, error) {
	resp := binaryopsserver.BinaryOpResponse{
		Result: request.Left + request.Right,
	}
	return &resp, nil
}

func (server *additionServer) Subtract(ctx context.Context, request *binaryopsserver.BinaryOpRequest) (*binaryopsserver.BinaryOpResponse, error) {
	resp := binaryopsserver.BinaryOpResponse{
		Result: request.Left - request.Right,
	}
	return &resp, nil
}
