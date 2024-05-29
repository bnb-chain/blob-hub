// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"

	"github.com/bnb-chain/blob-hub/client"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/swag"

	"github.com/bnb-chain/blob-hub/cache"
	"github.com/bnb-chain/blob-hub/config"
	syncerdb "github.com/bnb-chain/blob-hub/db"
	"github.com/bnb-chain/blob-hub/external"
	blobproto "github.com/bnb-chain/blob-hub/proto"
	"github.com/bnb-chain/blob-hub/restapi/handlers"
	"github.com/bnb-chain/blob-hub/restapi/operations"
	"github.com/bnb-chain/blob-hub/restapi/operations/blob"
	"github.com/bnb-chain/blob-hub/service"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

//go:generate swagger generate server --target ../../blob-syncer --name BlobHub --spec ../swagger.yaml --principal interface{}

var cliOpts = struct {
	ConfigFilePath string `short:"c" long:"config-path" description:"Config path" default:""`
}{}

func configureFlags(api *operations.BlobHubAPI) {
	param := swag.CommandLineOptionsGroup{
		ShortDescription: "config",
		Options:          &cliOpts,
	}
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{param}
}

func configureAPI(api *operations.BlobHubAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	api.BlobGetBlobSidecarsByBlockNumHandler = blob.GetBlobSidecarsByBlockNumHandlerFunc(handlers.HandleGetBlobSidecars())
	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	go grpcServer()

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
	var (
		cfg      *config.ServerConfig
		cacheSvc cache.Cache
		err      error
	)
	configFilePath := cliOpts.ConfigFilePath
	if configFilePath == "" {
		configFilePath = os.Getenv(config.EnvVarConfigFilePath)
	}
	cfg = config.ParseServerConfigFromFile(configFilePath)
	if cfg == nil {
		panic("failed to get configuration")
	}
	cfg.Validate()
	db := config.InitDBWithConfig(&cfg.DBConfig, false)
	blobDB := syncerdb.NewBlobSvcDB(db)
	bundleClient, err := external.NewBundleClient(cfg.BundleServiceEndpoints[0])
	if err != nil {
		panic(err)
	}

	switch cfg.CacheConfig.CacheType {
	case "local":
		cacheSvc, err = cache.NewLocalCache(cfg.CacheConfig.GetCacheSize())
		if err != nil {
			panic(err)
		}
	default:
		panic("currently only local cache is support.")
	}
	service.BlobSvc = service.NewBlobService(blobDB, bundleClient, cacheSvc, cfg)

}

func grpcServer() {
	lis, err := net.Listen("tcp", "0.0.0.0:9000")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()
	// Attach the Blob service to the server
	blobproto.RegisterBlobServiceServer(s, &client.BlobServer{})
	// Serve gRPC server
	log.Println("Serving gRPC on 0.0.0.0:9000")
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	maxMsgSize := 1024 * 1024 * 20
	// Create a client connection to the gRPC server we just started
	// This is where the gRPC-Gateway proxies the requests
	conn, err := grpc.DialContext(
		context.Background(),
		"0.0.0.0:8080",
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}
	gwmux := grpcruntime.NewServeMux()
	// Register User Service
	err = blobproto.RegisterBlobServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}
	gwServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", os.Getenv("server_port")),
		Handler: gwmux,
	}
	log.Println(fmt.Sprintf("Serving gRPC-Gateway on %s:%s", os.Getenv("server_host"), os.Getenv("server_port")))
	log.Fatalln(gwServer.ListenAndServe())
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
