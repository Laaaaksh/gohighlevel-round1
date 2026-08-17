package health

import "go.mongodb.org/mongo-driver/v2/mongo"

type IModule interface {
	GetCore() ICore
	GetHandler() *HTTPHandler
}

type module struct {
	core    ICore
	handler *HTTPHandler
}

func (m *module) GetCore() ICore           { return m.core }
func (m *module) GetHandler() *HTTPHandler { return m.handler }

// NewModule wires the module from the client it is handed, every call. It is
// a var, not a func, so a test can swap in a fake module and restore the
// original in TearDownTest per the testing standard.
var NewModule = func(mongoClient *mongo.Client) IModule {
	core := NewCore(mongoClient)
	handler := NewHTTPHandler(core)
	return &module{core: core, handler: handler}
}
