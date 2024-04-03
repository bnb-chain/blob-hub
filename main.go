package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bnb-chain/blob-syncer/config"
	syncerdb "github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/syncer"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

func initFlags() {
	flag.String(config.FlagConfigPath, "", "config file path")
	flag.String(config.FlagConfigType, "", "config type, local_private_key or aws_private_key")
	flag.String(config.FlagConfigAwsRegion, "", "aws region")
	flag.String(config.FlagConfigAwsSecretKey, "", "aws secret key")
	flag.String(config.FlagConfigPrivateKey, "", "blob-syncer private key")
	flag.String(config.FlagConfigBlsPrivateKey, "", "blob-syncer bls private key")
	flag.String(config.FlagConfigDbPass, "", "blob-syncer db password")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}
}

func printUsage() {
	fmt.Print("usage: ./blob-syncer --config-type local --config-path configFile\n")
	fmt.Print("usage: ./blob-syncer --config-type aws --aws-region awsRegin --aws-secret-key awsSecretKey\n")
}

func main() {
	var (
		cfg                        *config.Config
		configType, configFilePath string
	)
	initFlags()
	configType = viper.GetString(config.FlagConfigType)
	configType = "local"
	if configType == "" {
		configType = os.Getenv(config.ConfigType)
	}
	if configType != config.AWSConfig && configType != config.LocalConfig {
		printUsage()
		return
	}
	if configType == config.AWSConfig {
		awsSecretKey := viper.GetString(config.FlagConfigAwsSecretKey)
		if awsSecretKey == "" {
			printUsage()
			return
		}
		awsRegion := viper.GetString(config.FlagConfigAwsRegion)
		if awsRegion == "" {
			printUsage()
			return
		}
		configContent, err := config.GetSecret(awsSecretKey, awsRegion)
		if err != nil {
			fmt.Printf("get aws config error, err=%s", err.Error())
			return
		}
		cfg = config.ParseConfigFromJson(configContent)
	} else {
		configFilePath = viper.GetString(config.FlagConfigPath)
		configFilePath = "config/config.json"
		if configFilePath == "" {
			configFilePath = os.Getenv(config.ConfigFilePath)
			if configFilePath == "" {
				printUsage()
				return
			}
		}
		cfg = config.ParseConfigFromFile(configFilePath)
	}
	if cfg == nil {
		panic("failed to get configuration")
	}
	logging.InitLogger(&cfg.LogConfig)

	username := cfg.DBConfig.Username
	password := viper.GetString(config.FlagConfigDbPass)
	if password == "" {
		password = os.Getenv(config.ConfigDBPass)
		if password == "" {
			password = getDBPass(&cfg.DBConfig)
		}
	}
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,          // Disable color
		},
	)
	var db *gorm.DB
	var err error
	var dialector gorm.Dialector

	if cfg.DBConfig.Dialect == config.DBDialectMysql {
		url := cfg.DBConfig.Url
		dbPath := fmt.Sprintf("%s:%s@%s", username, password, url)
		dialector = mysql.Open(dbPath)
	} else if cfg.DBConfig.Dialect == config.DBDialectSqlite3 {
		dialector = sqlite.Open(cfg.DBConfig.Url)
	} else {
		panic(fmt.Sprintf("unexpected DB dialect %s", cfg.DBConfig.Dialect))
	}
	db, err = gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic(fmt.Sprintf("open db error, err=%s", err.Error()))
	}
	dbConfig, err := db.DB()
	if err != nil {
		panic(err)
	}

	dbConfig.SetMaxIdleConns(cfg.DBConfig.MaxIdleConns)
	dbConfig.SetMaxOpenConns(cfg.DBConfig.MaxOpenConns)

	syncerdb.InitTables(db)

	blobDB := syncerdb.NewBlobSvcDB(db)

	bs := syncer.NewBlobSyncer(blobDB, cfg)
	go bs.StartLoop()
	select {}
}

func getDBPass(cfg *config.DBConfig) string {
	if cfg.KeyType == config.KeyTypeAWSPrivateKey {
		result, err := config.GetSecret(cfg.AWSSecretName, cfg.AWSRegion)
		if err != nil {
			panic(err)
		}
		type DBPass struct {
			DbPass string `json:"db_pass"`
		}
		var dbPassword DBPass
		err = json.Unmarshal([]byte(result), &dbPassword)
		if err != nil {
			panic(err)
		}
		return dbPassword.DbPass
	}
	return cfg.Password
}
