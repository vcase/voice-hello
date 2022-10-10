package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	hellogen "github.com/vcase/voice-hello/generated/grpc/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	keyServerAddr = "serverAddr"
)

var (
	// grpc cmd line options and state

	grpcServerEndpoint string
	grpcShutdownTimout time.Duration

	grpcServer *grpc.Server

	// rest cmd line options

	restServerEndpoint string
	restShutdownTimout time.Duration
	restServer         *http.Server

	// swagger-ui cmd line options

	swaggerUiServerEndpoint string
	swaggerUiShutdownTimout time.Duration

	//go:embed generated/resources/swagger-ui
	swaggerRootFS   embed.FS
	swaggerUiServer *http.Server

	// pprof cmd line options

	pprofServerEndpoint string
	pprofShutdownTimout time.Duration
	pprofServer         *http.Server
)

func main() {
	var err error

	ctx := context.Background()

	// Options

	flag.StringVar(&grpcServerEndpoint, "grpc-endpoint", ":9090", "grpc server endpoint")
	flag.DurationVar(&grpcShutdownTimout, "grpc-shutdown-timout", 10*time.Second, "grpc server shutdown timeout")

	flag.StringVar(&restServerEndpoint, "rest-endpoint", ":8080", "rest server endpoint")
	flag.DurationVar(&restShutdownTimout, "rest-shutdown-timout", 10*time.Second, "rest server shutdown timeout")

	flag.StringVar(&swaggerUiServerEndpoint, "swagger-ui-endpoint", ":8081", "swagger-ui server endpoint")
	flag.DurationVar(&swaggerUiShutdownTimout, "swagger-ui-shutdown-timout", 10*time.Second, "swagger-ui server shutdown timeout")

	flag.StringVar(&pprofServerEndpoint, "pprof-endpoint", ":8082", "pprof server endpoint")
	flag.DurationVar(&pprofShutdownTimout, "pprof-shutdown-timout", 10*time.Second, "pprof server shutdown timeout")

	flag.Parse()

	// Start grpcServer and configure graceful stop
	gprcErrChan := make(chan error, 1)
	grpcServer, err = runGrpc(gprcErrChan)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdownGrpc()

	// Start restServer and configure graceful shutdown
	restErrChan := make(chan error, 1)
	restServer, err = runRest(restErrChan)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdownRest(ctx)

	// Start swaggerUiServer and configure graceful shutdown
	swaggerUiErrChan := make(chan error, 1)
	swaggerUiServer, err = runSwaggerUI(swaggerUiErrChan)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdownSwaggerUI(ctx)

	// Start pprofServer and configure graceful shutdown
	pprofErrChan := make(chan error, 1)
	pprofServer, err = runPProf(pprofErrChan)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdownPProf(ctx)

	// Wait for Cancel or Error
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-gprcErrChan:
		log.Printf("grpc server error: %v\n", err)
	case err := <-restErrChan:
		log.Printf("rest server error: %v\n", err)
	case err := <-swaggerUiErrChan:
		log.Printf("swagger-ui server error: %v\n", err)
	case err := <-pprofErrChan:
		log.Printf("pprof server error: %v\n", err)
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

	log.Printf("Starting grpc server at %s\n", grpcServerEndpoint)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			errChan <- err
		}
	}()

	return grpcServer, nil
}

func shutdownGrpc() {
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
}

func runRest(errChan chan<- error) (*http.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// File server for openapi specs (json)
	specsFS, err := fs.Sub(swaggerRootFS, "generated/resources/swagger-ui/openapi-specs")
	if err != nil {
		return nil, err
	}
	specsFileServer := http.FileServer(http.FS(specsFS))

	// grpc gateway server
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = hellogen.RegisterHelloHandlerFromEndpoint(ctx, gwMux, grpcServerEndpoint, opts)
	if err != nil {
		return nil, err
	}

	// Mux to route to both
	mux := http.NewServeMux()

	prefix := "/openapi-specs/"
	mux.Handle(prefix, http.StripPrefix(prefix, specsFileServer))
	mux.Handle("/", gwMux)

	// CORS support added by wrapping in handerFunc
	corsMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS")
		} else if strings.Contains(origin, "localhost") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == "OPTIONS" {
				return // done with cors OPTIONS request
			}
		} else {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(os.Stderr, "Request for invalid origin: '%s'\n", origin)
			return
		}
		mux.ServeHTTP(w, r)
	})

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	httpServer := &http.Server{
		Addr:    restServerEndpoint,
		Handler: corsMux,
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

func shutdownRest(ctx context.Context) {
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
}

func runSwaggerUI(errChan chan<- error) (*http.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	swaggerUiContent, err := fs.Sub(swaggerRootFS, "generated/resources/swagger-ui")
	if err != nil {

		return nil, fmt.Errorf("swagger-ui content subdirectory error: %v", err)
	}

	mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.FS(swaggerUiContent))))

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	httpServer := &http.Server{
		Addr:    swaggerUiServerEndpoint,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, keyServerAddr, l.Addr().String())
			return ctx
		},
	}

	log.Printf("Starting swagger-ui server at %s\n", swaggerUiServerEndpoint)
	go func() {
		defer cancel()
		err := httpServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}

	}()

	return httpServer, nil
}

func shutdownSwaggerUI(ctx context.Context) {
	if swaggerUiServer == nil {
		return
	}

	log.Printf("swagger-ui server shudown started")
	ctx, cancel := context.WithTimeout(ctx, swaggerUiShutdownTimout)
	defer cancel()
	err := swaggerUiServer.Shutdown(ctx)
	if err == nil {
		log.Printf("swagger-ui server shutdown completed")
	} else {
		log.Printf("swagger-ui server shutdown completion status: %v", err)
	}
}

func runPProf(errChan chan<- error) (*http.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	mux.HandleFunc("/pprof/", pprof.Index)
	mux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/pprof/profile", pprof.Profile)
	mux.HandleFunc("/pprof/symbol", pprof.Symbol)

	// Manually add support for paths linked to by index page at /pprof/
	mux.Handle("/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/pprof/block", pprof.Handler("block"))

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	httpServer := &http.Server{
		Addr:    pprofServerEndpoint,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, keyServerAddr, l.Addr().String())
			return ctx
		},
	}

	log.Printf("Starting pprof server at %s\n", pprofServerEndpoint)
	go func() {
		defer cancel()
		err := httpServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}

	}()

	return httpServer, nil
}

func shutdownPProf(ctx context.Context) {
	if swaggerUiServer == nil {
		return
	}

	log.Printf("swagger-ui server shudown started")
	ctx, cancel := context.WithTimeout(ctx, swaggerUiShutdownTimout)
	defer cancel()
	err := swaggerUiServer.Shutdown(ctx)
	if err == nil {
		log.Printf("swagger-ui server shutdown completed")
	} else {
		log.Printf("swagger-ui server shutdown completion status: %v", err)
	}
}
