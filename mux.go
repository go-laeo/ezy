package ezy

import (
	"context"
	"net/http"
	"net/url"
)

const (
	customANY = "ANY"
)

type routePathParam struct{}

var defaultNotFoundHandler HandlerFunc[Void] = func(ctx Context, _ *Void) error {
	ctx.WriteHeader(404)
	ctx.Write([]byte("not found"))
	return nil
}

type ServerMux struct {
	ctx      context.Context
	notfound http.Handler
	root     *Route
	prefix   string
}

func NewServerMux(ctx context.Context) *ServerMux {
	return &ServerMux{
		ctx:      ctx,
		root:     &Route{},
		notfound: defaultNotFoundHandler,
	}
}

var _ http.Handler = (*ServerMux)(nil)

func (sm *ServerMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cap := make(url.Values)
	r = r.WithContext(context.WithValue(sm.ctx, &routePathParam{}, cap))
	n := sm.root.Search(r.URL.Path, cap)
	if n != nil {
		fn, ok := n.hmap[r.Method]
		if ok {
			fn.ServeHTTP(w, r)
			return
		}

		fn, ok = n.hmap[customANY]
		if ok {
			fn.ServeHTTP(w, r)
			return
		}
	}

	sm.notfound.ServeHTTP(w, r)
}

func (sm *ServerMux) Get(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodGet, h)
}

func (sm *ServerMux) Post(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodPost, h)
}

func (sm *ServerMux) Put(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodPut, h)
}

func (sm *ServerMux) Delete(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodDelete, h)
}

func (sm *ServerMux) Patch(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodPatch, h)
}

func (sm *ServerMux) Options(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodOptions, h)
}

func (sm *ServerMux) Head(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, http.MethodHead, h)
}

func (sm *ServerMux) Any(path string, h http.Handler) {
	sm.root.Insert(sm.prefix+path, customANY, h)
}

func (sm *ServerMux) Group(prefix string, fn func(sm *ServerMux)) {
	prev := sm.prefix
	sm.prefix = sm.prefix + prefix
	fn(sm)
	sm.prefix = prev
}
