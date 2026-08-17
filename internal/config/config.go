// Package config loads typed configuration from config/default.toml,
// overlaid by an environment-specific TOML file, overlaid by a small set of
// well-known environment variables (themselves optionally seeded from a
// local .env file). Nothing in the service reads an env var or a config
// value by string key outside this package.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	configFileType                    = "toml"
	configDirName                     = "config"
	configDirRelativeOneUp            = "../config"
	configDirRelativeTwoUp            = "../../config"
	dotEnvFileName                    = ".env"
	dotEnvRelativeOneUp               = "../.env"
	dotEnvRelativeTwoUp               = "../../.env"
	defaultConfigName                 = "default"
	defaultAppEnv                     = "dev"
	defaultMongoConnectTimeoutSeconds = 10

	envAppEnv   = "APP_ENV"
	envPort     = "PORT"
	envMongoURI = "MONGO_URI"

	viperKeyServerPort = "server.port"
	viperKeyMongoURI   = "mongo.uri"
)

var (
	ErrLoadDefaultConfig = errors.New("failed to load default config")
	ErrLoadEnvConfig     = errors.New("failed to load environment config")
	ErrUnmarshalConfig   = errors.New("failed to unmarshal config")
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port          string `mapstructure:"port"`
	AllowedOrigin string `mapstructure:"allowed_origin"`
}

// MongoConfig holds MongoDB connection settings.
type MongoConfig struct {
	URI                   string `mapstructure:"uri"`
	Database              string `mapstructure:"database"`
	ConnectTimeoutSeconds int    `mapstructure:"connect_timeout_seconds"`
}

// Config is the root typed configuration for the service.
type Config struct {
	Env    string       `mapstructure:"env"`
	Server ServerConfig `mapstructure:"server"`
	Mongo  MongoConfig  `mapstructure:"mongo"`
}

// loadDotEnv seeds the process environment from a local .env file so
// `cp .env.example .env` works without exporting anything by hand. It is
// deliberately best-effort: a missing file is normal in production and CI,
// where real environment variables are supplied instead. godotenv never
// overwrites a variable that is already set, so a real shell variable always
// wins over the file.
func loadDotEnv() {
	// Same depth problem as the config search paths below: go test runs from
	// the package directory, not the repo root.
	for _, path := range []string{dotEnvFileName, dotEnvRelativeOneUp, dotEnvRelativeTwoUp} {
		if err := godotenv.Load(path); err == nil {
			return
		}
	}
}

// Load reads a local .env file if present, then config/default.toml, merges
// config/<APP_ENV>.toml on top, then applies the PORT and MONGO_URI
// environment variable overrides the brief requires explicitly.
func Load() (Config, error) {
	loadDotEnv()

	v, err := readConfigFiles(resolveEnv())
	if err != nil {
		return Config{}, err
	}
	applyEnvOverrides(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrUnmarshalConfig, err)
	}
	if cfg.Mongo.ConnectTimeoutSeconds == 0 {
		cfg.Mongo.ConnectTimeoutSeconds = defaultMongoConnectTimeoutSeconds
	}
	return cfg, nil
}

func resolveEnv() string {
	if env := os.Getenv(envAppEnv); env != "" {
		return env
	}
	return defaultAppEnv
}

// readConfigFiles reads config/default.toml, then merges config/<env>.toml on
// top of it. A missing environment file is fine - default.toml alone is a
// complete configuration.
func readConfigFiles(env string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType(configFileType)
	// go test sets the working directory to the package under test, not the
	// repo root, so list every depth a test package might run from.
	v.AddConfigPath(configDirName)
	v.AddConfigPath(configDirRelativeOneUp)
	v.AddConfigPath(configDirRelativeTwoUp)

	v.SetConfigName(defaultConfigName)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLoadDefaultConfig, err)
	}

	v.SetConfigName(env)
	if err := v.MergeInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("%w: %v", ErrLoadEnvConfig, err)
		}
	}
	return v, nil
}

// applyEnvOverrides lets the two variables the brief calls out explicitly win
// over whatever the TOML files resolved to.
func applyEnvOverrides(v *viper.Viper) {
	if port := os.Getenv(envPort); port != "" {
		v.Set(viperKeyServerPort, port)
	}
	if mongoURI := os.Getenv(envMongoURI); mongoURI != "" {
		v.Set(viperKeyMongoURI, mongoURI)
	}
}
