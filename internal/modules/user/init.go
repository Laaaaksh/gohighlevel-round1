package user

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// IModule exposes the user module's wired dependencies to boot.go and, via
// GetCore, to any other module that needs to validate a userId.
type IModule interface {
	GetCore() ICore
	GetHandler() *HTTPHandler
	GetRepository() IRepository
}

type module struct {
	core       ICore
	handler    *HTTPHandler
	repository IRepository
}

func (m *module) GetCore() ICore             { return m.core }
func (m *module) GetHandler() *HTTPHandler   { return m.handler }
func (m *module) GetRepository() IRepository { return m.repository }

// NewModule wires the module from the collection it is handed, every call.
// It is a var, not a func, so a test can swap in a fake module and restore
// the original in TearDownTest per the testing standard.
var NewModule = func(collection *mongo.Collection) IModule {
	repo := NewRepository(collection)
	core := NewCore(repo)
	handler := NewHTTPHandler(core)
	return &module{core: core, handler: handler, repository: repo}
}
