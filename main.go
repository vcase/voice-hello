package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	hellogen "github.com/vcase/voice-hello/generated/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	keyServerAddr = "serverAddr"
)

var (
	// cmd line options
	grpcServerEndpoint string
	restServerEndpoint string

	// grpc server state
	grpcShutdownTimout time.Duration
	grpcServer         *grpc.Server

	// rest server state
	restShutdownTimout time.Duration
	restServer         *http.Server
)

func main() {
	var err error

	ctx := context.Background()

	// Options

	flag.StringVar(&grpcServerEndpoint, "grpc-endpoint", ":9090", "grpc server endpoint")
	flag.StringVar(&restServerEndpoint, "rest-endpoint", ":8080", "rest server endpoint")

	flag.DurationVar(&grpcShutdownTimout, "grpc-shutdown-timout", 10*time.Second, "grpc server shutdown timeout")
	flag.DurationVar(&restShutdownTimout, "rest-shutdown-timout", 10*time.Second, "rest server shutdown timeout")

	flag.Parse()

	// Start grpcServer and configure graceful stop
	gprcErrChan := make(chan error, 1)
	grpcServer, err = runGrpc(gprcErrChan)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if grpcServer == nil {
			return
		}

		log.Println("grpc server shutdown started")
		stopCh := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			stopCh <- struct{}{}
		}()

		select {
		case <-stopCh:
			log.Println("grpc server shutdown completed")
		case <-time.After(grpcShutdownTimout):
			log.Println("grpc server shutdown timed out")
		}
	}()

	// Start restServer and configure graceful shutdown
	restErrChan := make(chan error, 1)
	restServer, err = runRest(restErrChan)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if restServer == nil {
			return
		}

		log.Printf("rest server shudown started")
		ctx, cancel := context.WithTimeout(ctx, restShutdownTimout)
		defer cancel()
		err := restServer.Shutdown(ctx)
		if err == nil {
			log.Printf("rest server shutdown completed")
		} else {
			log.Printf("rest server shutdown completion status: %v", err)
		}
	}()

	// Wait for Cancel or Error
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-gprcErrChan:
		log.Printf("grpc server error: %v\n", err)
	case err := <-restErrChan:
		log.Printf("rest server error: %v\n", err)
	case err := <-sigChan:
		log.Printf("shutdown signal received: %v\n", err)
	}
}

func runGrpc(errChan chan<- error) (*grpc.Server, error) {

	lis, err := net.Listen("tcp", grpcServerEndpoint)
	if err != nil {
		return nil, err
	}

	server := &helloServer{}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	hellogen.RegisterHelloServer(grpcServer, server)

	log.Printf("Starting grpc server at %s\n", restServerEndpoint)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			errChan <- err
		}
	}()

	return grpcServer, nil
}

func runRest(errChan chan<- error) (*http.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := hellogen.RegisterHelloHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts)
	if err != nil {
		return nil, err
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	httpServer := &http.Server{
		Addr:    restServerEndpoint,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, keyServerAddr, l.Addr().String())
			return ctx
		},
	}

	log.Printf("Starting rest server at %s\n", restServerEndpoint)
	go func() {
		defer cancel()
		err := httpServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}

	}()

	return httpServer, nil
}
