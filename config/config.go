package config

import (
	"encoding/json"
	"fmt"
	"github.com/bnb-chain/blob-syncer/cache"
	"os"
)

type Config struct {
	LogConfig    LogConfig    `json:"log_config"`
	DBConfig     DBConfig     `json:"db_config"`
	SyncerConfig SyncerConfig `json:"syncer_config"`
	ServerConfig ServerConfig `json:"server_config"`
	CacheConfig  CacheConfig  `json:"cache_config"`
}

type SyncerConfig struct {
	BucketName             string   `json:"bucket_name"`              // BucketName is the identifier of bucket on Greenfield that store blob
	StartSlot              uint64   `json:"start_slot"`               // StartSlot is used to init the syncer which slot of beacon chain to synced from
	BundleServiceEndpoints []string `json:"bundle_service_endpoints"` // BundleServiceEndpoints is a list of bundle service address
	BeaconRPCAddrs         []string `json:"beacon_rpc_addrs"`         // BeaconRPCAddrs is a list of beacon chain RPC address
	ETHRPCAddrs            []string `json:"eth_rpc_addrs"`
	TempFilePath           string   `json:"temp_file_path"` // TempFilePath used to create file for every blob.
	PrivateKey             string   `json:"private_key"`
}

type ServerConfig struct {
	BucketName         string   `json:"bucket_name"`
	BundleServiceAddrs []string `json:"bundle_service_addrs"`
}

type CacheConfig struct {
	CacheType string `json:"cache_type"`
	URL       string `json:"url"`
	CacheSize uint64 `json:"cache_size"`
}

func (c *CacheConfig) GetCacheSize() uint64 {
	if c.CacheSize != 0 {
		return c.CacheSize
	}
	return cache.DefaultCacheSize
}

type DBConfig struct {
	Dialect      string `json:"dialect"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Url          string `json:"url"`
	MaxIdleConns int    `json:"max_idle_conns"`
	MaxOpenConns int    `json:"max_open_conns"`
}

func (cfg *DBConfig) Validate() {
	if cfg.Dialect != DBDialectMysql && cfg.Dialect != DBDialectSqlite3 {
		panic(fmt.Sprintf("only %s and %s supported", DBDialectMysql, DBDialectSqlite3))
	}
	if cfg.Dialect == DBDialectMysql && (cfg.Username == "" || cfg.Url == "") {
		panic("db config is not correct, missing username and/or url")
	}
	if cfg.MaxIdleConns == 0 || cfg.MaxOpenConns == 0 {
		panic("db connections is not correct")
	}
}

type LogConfig struct {
	Level                        string `json:"level"`
	Filename                     string `json:"filename"`
	MaxFileSizeInMB              int    `json:"max_file_size_in_mb"`
	MaxBackupsOfLogFiles         int    `json:"max_backups_of_log_files"`
	MaxAgeToRetainLogFilesInDays int    `json:"max_age_to_retain_log_files_in_days"`
	UseConsoleLogger             bool   `json:"use_console_logger"`
	UseFileLogger                bool   `json:"use_file_logger"`
	Compress                     bool   `json:"compress"`
}

func (cfg *LogConfig) Validate() {
	if cfg.UseFileLogger {
		if cfg.Filename == "" {
			panic("filename should not be empty if use file logger")
		}
		if cfg.MaxFileSizeInMB <= 0 {
			panic("max_file_size_in_mb should be larger than 0 if use file logger")
		}
		if cfg.MaxBackupsOfLogFiles <= 0 {
			panic("max_backups_off_log_files should be larger than 0 if use file logger")
		}
	}
}

func ParseConfigFromJson(content string) *Config {
	var config Config
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		panic(err)
	}
	//config.Validate()
	return &config
}

func ParseConfigFromFile(filePath string) *Config {
	bz, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var config Config
	if err := json.Unmarshal(bz, &config); err != nil {
		panic(err)
	}
	//config.Validate()
	return &config
}
