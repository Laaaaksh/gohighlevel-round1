package timeline

// IModule exposes the timeline module's wired dependencies to boot.go.
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

// NewModule wires the module from the post and follow readers it is
// handed, every call. It is a var, not a func, so a test can swap in a fake
// module and restore the original in TearDownTest per the testing standard.
var NewModule = func(posts PostReader, follows FollowReader) IModule {
	core := NewCore(posts, follows)
	handler := NewHTTPHandler(core)
	return &module{core: core, handler: handler}
}
