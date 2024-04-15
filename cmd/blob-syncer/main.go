package main

import (
	"flag"
	"github.com/bnb-chain/blob-syncer/metrics"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bnb-chain/blob-syncer/config"
	syncerdb "github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/syncer"
)

func initFlags() {
	flag.String(config.FlagConfigPath, "", "config file path")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}
}

func main() {
	var (
		cfg            *config.SyncerConfig
		configFilePath string
	)
	initFlags()
	configFilePath = viper.GetString(config.FlagConfigPath)
	if configFilePath == "" {
		configFilePath = os.Getenv(config.EnvVarConfigFilePath)
	}
	configFilePath = "config/local/config-syncer.json"
	cfg = config.ParseSyncerConfigFromFile(configFilePath)
	if cfg == nil {
		panic("failed to get configuration")
	}
	cfg.Validate()
	logging.InitLogger(&cfg.LogConfig)
	db := config.InitDBWithConfig(&cfg.DBConfig, true)
	blobDB := syncerdb.NewBlobSvcDB(db)
	bs := syncer.NewBlobSyncer(blobDB, cfg)
	go bs.StartLoop()

	if cfg.MetricsConfig.Enable {
		if cfg.MetricsConfig.HttpAddress == "" {
			cfg.MetricsConfig.HttpAddress = metrics.DefaultMetricsAddress
		}
		metric := metrics.NewMetrics(cfg.MetricsConfig.HttpAddress)
		go metric.Start()
	}

	select {}
}
