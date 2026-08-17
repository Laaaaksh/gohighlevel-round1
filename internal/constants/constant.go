// Package constants holds values shared across more than one module.
// Values used by only a single module or package belong closer to their
// use - see internal/modules/<feature>/entities/constant.go and per-package
// constants.go files.
package constants

const (
	ServiceName = "gohighlevel-round1-api"
	APIVersion  = "v1"
)

const (
	HeaderRequestID   = "X-Request-ID"
	HeaderContentType = "Content-Type"
	ContentTypeJSON   = "application/json"
)

const (
	EnvDevelopment = "dev"
	EnvTest        = "test"
	EnvProduction  = "production"
)
