package item

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// IModule exposes the item module's wired dependencies to boot.go.
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

// mod is the module singleton. NewModule is a var, not a func, so tests can
// swap it out; reset it in SetupTest/TearDownTest per the testing standard.
var mod IModule

var NewModule = func(collection *mongo.Collection) IModule {
	if mod == nil {
		repo := NewRepository(collection)
		core := NewCore(repo)
		handler := NewHTTPHandler(core)
		mod = &module{core: core, handler: handler, repository: repo}
	}
	return mod
}
