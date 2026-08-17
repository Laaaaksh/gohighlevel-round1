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

var mod IModule

var NewModule = func(mongoClient *mongo.Client) IModule {
	if mod == nil {
		core := NewCore(mongoClient)
		handler := NewHTTPHandler(core)
		mod = &module{core: core, handler: handler}
	}
	return mod
}
