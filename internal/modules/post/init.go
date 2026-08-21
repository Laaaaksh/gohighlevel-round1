package post

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// IModule exposes the post module's wired dependencies to boot.go and, via
// GetCore, to the timeline module for the paginated author-fetch it reuses.
type IModule interface {
	GetCore() ICore
	GetHandler() *HTTPHandler
	GetRepository() IRepository
	// GetCoreConcrete exposes the concrete *Core (not just ICore) so the
	// timeline module can call its ListByAuthors, which is not part of
	// ICore - the post module's own HTTP surface only ever needs
	// CreatePost/ListByUser.
	GetCoreConcrete() *Core
}

type module struct {
	core       *Core
	handler    *HTTPHandler
	repository IRepository
}

func (m *module) GetCore() ICore             { return m.core }
func (m *module) GetHandler() *HTTPHandler   { return m.handler }
func (m *module) GetRepository() IRepository { return m.repository }

func (m *module) GetCoreConcrete() *Core { return m.core }

// NewModule wires the module from the collection and user checker it is
// handed, every call. It is a var, not a func, so a test can swap in a fake
// module and restore the original in TearDownTest per the testing standard.
var NewModule = func(collection *mongo.Collection, users UserChecker) IModule {
	repo := NewRepository(collection)
	core := NewCore(repo, users)
	handler := NewHTTPHandler(core)
	return &module{core: core, handler: handler, repository: repo}
}
