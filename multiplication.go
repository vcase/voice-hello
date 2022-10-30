package main

import (
	"context"

	binaryopsserver "github.com/vcase/voice-servicetemplate/generated/grpc/service-template/binaryops"
)

type multiplicationServer struct {
	binaryopsserver.UnimplementedMultiplicationServer
}

func (server *multiplicationServer) Multiply(ctx context.Context, request *binaryopsserver.BinaryOpRequest) (*binaryopsserver.BinaryOpResponse, error) {
	resp := binaryopsserver.BinaryOpResponse{
		Result: request.Left * request.Right,
	}
	return &resp, nil
}

func (server *multiplicationServer) Divide(ctx context.Context, request *binaryopsserver.BinaryOpRequest) (*binaryopsserver.BinaryOpResponse, error) {
	resp := binaryopsserver.BinaryOpResponse{
		Result: request.Left / request.Right,
	}
	return &resp, nil
}
