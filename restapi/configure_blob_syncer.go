// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"net/http"
	"os"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/swag"

	"github.com/bnb-chain/blob-syncer/cache"
	"github.com/bnb-chain/blob-syncer/config"
	syncerdb "github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/external"
	"github.com/bnb-chain/blob-syncer/restapi/handlers"
	"github.com/bnb-chain/blob-syncer/restapi/operations"
	"github.com/bnb-chain/blob-syncer/restapi/operations/blob"
	"github.com/bnb-chain/blob-syncer/service"
)

//go:generate swagger generate server --target ../../blob-syncer --name BlobSyncer --spec ../swagger.yaml --principal interface{}

var cliOpts = struct {
	ConfigFilePath string `short:"c" long:"config-path" description:"Config path" default:""`
}{}

func configureFlags(api *operations.BlobSyncerAPI) {
	param := swag.CommandLineOptionsGroup{
		ShortDescription: "config",
		Options:          &cliOpts,
	}
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{param}
}

func configureAPI(api *operations.BlobSyncerAPI) http.Handler {
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
