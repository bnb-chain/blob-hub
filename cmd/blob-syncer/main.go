package main

import (
	"flag"
	"fmt"
	"github.com/bnb-chain/blob-syncer/config"
	syncerdb "github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/syncer"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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

func printUsage() {
	fmt.Print("usage: ./blob-syncer --config-path configFile\n")
}

func main() {
	var (
		cfg            *config.SyncerConfig
		configFilePath string
	)
	initFlags()
	configFilePath = viper.GetString(config.FlagConfigPath)

	configFilePath = "config/local/config-syncer.json" //todo
	if configFilePath == "" {
		printUsage()
		return
	}
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
	select {}
}
