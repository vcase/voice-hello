package main

import (
	"flag"
)

var (
	// command-line options:
	// gRPC server endpoint
	grpcServerEndpoint = flag.String("grpc-server-endpoint", "localhost:9090", "gRPC server endpoint")
)

// func runRest() error {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	// Register gRPC server endpoint
// 	// Note: Make sure the gRPC server is running properly and accessible
// 	mux := runtime.NewServeMux()
// 	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
// 	err := hellogen.RegisterHelloHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
// 	if err != nil {
// 		return err
// 	}

// 	// Start HTTP server (and proxy calls to gRPC server endpoint)
// 	return http.ListenAndServe(":8081", mux)
// }
