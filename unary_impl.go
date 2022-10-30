package main

import (
	"context"

	unaryopsserver "github.com/vcase/voice-servicetemplate/generated/grpc/service-template/unaryops"
)

type unaryServer struct {
	unaryopsserver.UnimplementedUnaryOpsServer
}

func (server *unaryServer) Abs(ctx context.Context, request *unaryopsserver.UnaryOpRequest) (*unaryopsserver.UnaryOpResponse, error) {
	result := request.Operand
	if result < 0 {
		result = -result
	}
	resp := unaryopsserver.UnaryOpResponse{
		Result: result,
	}
	return &resp, nil
}
