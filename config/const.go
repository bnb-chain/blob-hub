package config

const (
	FlagConfigPath          = "config-path"
	FlagConfigType          = "config-type"
	FlagConfigAwsRegion     = "aws-region"
	FlagConfigAwsSecretKey  = "aws-secret-key"
	FlagConfigPrivateKey    = "private-key"
	FlagConfigBlsPrivateKey = "bls-private-key"
	FlagConfigDbPass        = "db-pass"

	DBDialectMysql   = "mysql"
	DBDialectSqlite3 = "sqlite3"

	LocalConfig            = "local"
	AWSConfig              = "aws"
	KeyTypeLocalPrivateKey = "local_private_key"
	KeyTypeAWSPrivateKey   = "aws_private_key"

	ConfigType     = "CONFIG_TYPE"
	ConfigFilePath = "CONFIG_FILE_PATH"
	ConfigDBPass   = "DB_PASS"
)
