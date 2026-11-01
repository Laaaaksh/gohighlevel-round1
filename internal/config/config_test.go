// External test package: Load is the whole public surface of internal/config,
// so the tests exercise it exactly as cmd/api and cmd/seed do.
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
)

const (
	envAppEnv   = "APP_ENV"
	envPort     = "PORT"
	envMongoURI = "MONGO_URI"

	configDirName   = "config"
	defaultTOMLName = "default.toml"
	dotEnvName      = ".env"

	testTOMLPort      = "8080"
	testDotEnvPort    = "9090"
	testShellEnvPort  = "7070"
	testTOMLMongoURI  = "mongodb://localhost:27017"
	testShellMongoURI = "mongodb://localhost:27018"

	testDirPerm  = 0o755
	testFilePerm = 0o644

	testDefaultTOML = `env = "dev"

[server]
port = "8080"
allowed_origin = "http://localhost:3000"

[mongo]
uri = "mongodb://localhost:27017"
database = "gohighlevel_round1"
connect_timeout_seconds = 10
`
	testDotEnvContents = "PORT=9090\nMONGO_URI=mongodb://localhost:27017\n"
)

type ConfigTestSuite struct {
	suite.Suite
	savedEnv map[string]string
	workDir  string
}

// SetupTest isolates each test in its own working directory holding a minimal
// config/default.toml, and clears the env vars Load consults - godotenv sets
// them on the real process environment, so they would otherwise leak between
// tests and mask the "shell wins" behaviour.
func (s *ConfigTestSuite) SetupTest() {
	s.savedEnv = make(map[string]string)
	for _, key := range []string{envAppEnv, envPort, envMongoURI} {
		s.savedEnv[key] = os.Getenv(key)
		s.Require().NoError(os.Unsetenv(key))
	}

	s.workDir = s.T().TempDir()
	configDir := filepath.Join(s.workDir, configDirName)
	s.Require().NoError(os.MkdirAll(configDir, testDirPerm))
	s.Require().NoError(os.WriteFile(
		filepath.Join(configDir, defaultTOMLName), []byte(testDefaultTOML), testFilePerm,
	))
	s.T().Chdir(s.workDir)
}

func (s *ConfigTestSuite) TearDownTest() {
	for key, value := range s.savedEnv {
		if value == "" {
			s.Require().NoError(os.Unsetenv(key))
			continue
		}
		s.Require().NoError(os.Setenv(key, value))
	}
}

func (s *ConfigTestSuite) writeDotEnv() {
	s.Require().NoError(os.WriteFile(
		filepath.Join(s.workDir, dotEnvName), []byte(testDotEnvContents), testFilePerm,
	))
}

func (s *ConfigTestSuite) TestLoadAppliesDotEnvFile() {
	s.writeDotEnv()

	cfg, err := config.Load()

	s.Require().NoError(err)
	s.Equal(testDotEnvPort, cfg.Server.Port)
	s.Equal(testTOMLMongoURI, cfg.Mongo.URI)
	s.Equal(testDotEnvPort, os.Getenv(envPort))
}

func (s *ConfigTestSuite) TestLoadPrefersShellEnvOverDotEnvFile() {
	s.writeDotEnv()
	s.Require().NoError(os.Setenv(envPort, testShellEnvPort))
	s.Require().NoError(os.Setenv(envMongoURI, testShellMongoURI))

	cfg, err := config.Load()

	s.Require().NoError(err)
	s.Equal(testShellEnvPort, cfg.Server.Port)
	s.Equal(testShellMongoURI, cfg.Mongo.URI)
	s.Equal(testShellEnvPort, os.Getenv(envPort))
}

func (s *ConfigTestSuite) TestLoadSucceedsWithoutDotEnvFile() {
	cfg, err := config.Load()

	s.Require().NoError(err)
	s.Equal(testTOMLPort, cfg.Server.Port)
	s.Equal(testTOMLMongoURI, cfg.Mongo.URI)
	s.Empty(os.Getenv(envPort))
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
