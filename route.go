package pi

import (
	"bytes"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const (
	dynamic  = ':'
	wildcard = '*'
	anyone   = "*"
)

type Route interface {
	// Search try to match `route`, and stores
	// path params to `captured`, if nothing
	// found, it will returns nil.
	Search(route string, captured url.Values) Route

	// Invoke calls corresponding HandlerFunc which
	// registered for ctx.Method().
	Invoke(ctx Context) bool

	Name(name string) Route

	// // From generates a *url.URL for route `name`,
	// // if nothing found, it will returns nil.
	// From(name string, v url.Values) *url.URL

	// For registers `h` for `method` on the route path.
	For(method string, h HandlerFunc) Route
	Get(h HandlerFunc) Route
	Post(h HandlerFunc) Route
	Put(h HandlerFunc) Route
	Delete(h HandlerFunc) Route
	Patch(h HandlerFunc) Route
	Options(h HandlerFunc) Route
	Head(h HandlerFunc) Route
	Any(h HandlerFunc) Route
}

var _ Route = (*_route)(nil)

type _route struct {
	parent           *_route
	sub              map[string]*_route
	hmap             map[string]HandlerFunc
	named            map[string]*_route
	pattern          string
	placeholder      string
	name             string
	cc               []func(HandlerFunc) HandlerFunc
	hasDynamicChild  bool
	hasWildcardChild bool
}

func createRootRoute() *_route {
	return &_route{
		sub:   make(map[string]*_route),
		hmap:  make(map[string]HandlerFunc),
		named: make(map[string]*_route),
	}
}

func (p *_route) Search(route string, captured url.Values) Route {
	route = path.Clean(route)
	chunks := strings.Split(route, "/") // TODO: use index to identify a chunk instead of call Split().

	current := p
SEARCH:
	for i := 0; i < len(chunks); i++ {
		seg := chunks[i]

		next, ok := current.sub[seg]
		if ok {
			current = next
			continue
		}

		if current.hasDynamicChild {
			next, ok = current.sub[string(dynamic)]
			if ok {
				captured.Add(next.placeholder, seg)
				current = next // continues on dynamic route.
				continue
			}
		}

		if current.hasWildcardChild {
			next, ok = current.sub[string(wildcard)]
			if ok {
				captured.Add(next.placeholder, strings.Join(chunks[i:], "/"))
				// wildcard route should returns immediately.
				return next
			}
		}

		for current.parent != nil {
			if current.parent.hasDynamicChild {
				k := string(dynamic)
				next = current.parent.sub[k]
				if next != current {
					i--
					captured.Add(next.placeholder, chunks[i])
					current = next
					continue SEARCH
				}
			}
			if current.parent.hasWildcardChild {
				k := string(wildcard)
				next = current.parent.sub[k]
				if next != current {
					i--
					captured.Add(next.placeholder, strings.Join(chunks[i:], "/"))
					return next
				}
			}

			i--
			current = current.parent
		}

		return nil
	}

	return current
}

func (p *_route) Insert(route string, cc ...func(HandlerFunc) HandlerFunc) *_route {
	route = path.Clean(route)
	current := p
	for _, seg := range strings.Split(route, "/") {
		if current.sub == nil {
			current.sub = make(map[string]*_route)
		}

		next, ok := current.sub[seg]
		if ok {
			current = next
			continue
		}

		next = &_route{
			parent:      current,
			pattern:     seg,
			placeholder: seg,
		}

		if len(seg) > 0 {
			switch next.pattern[0] {
			case dynamic:
				current.hasDynamicChild = true
				next.placeholder = next.placeholder[1:]
				current.sub[string(dynamic)] = next
			case wildcard:
				current.hasWildcardChild = true
				next.placeholder = next.placeholder[1:]
				current.sub[string(wildcard)] = next
			}
		}

		current.sub[seg] = next
		current = next
	}

	if current.hmap == nil {
		current.hmap = make(map[string]HandlerFunc)
	}

	current.cc = append(current.cc, cc...)

	return current
}

func (p *_route) Invoke(ctx Context) bool {
	fn, ok := p.hmap[ctx.Method()]
	if !ok {
		fn, ok = p.hmap[anyone]
	}
	if !ok {
		return false
	}

	if err := fn(ctx); err != nil {
		switch v := err.(type) {
		case *Error:
			ctx.Code(v.Code)
			ctx.Json(v)
		default:
			ctx.Code(http.StatusInternalServerError)
			ctx.Json(NewError(http.StatusInternalServerError, err.Error()))
		}
	}

	return true
}

func (p *_route) Name(name string) Route {
	p.name = name
	root := p
	for root.parent != nil {
		root = root.parent
	}
	root.named[name] = p
	return p
}

func (p *_route) From(name string, v url.Values) []byte {
	root := p
	for root.parent != nil {
		root = root.parent
	}

	t, ok := root.named[name]
	if !ok {
		return nil
	}

	seg := make([]string, 0)
	for t != nil && t.pattern != "" {
		if v.Has(t.placeholder) {
			seg = append(seg, v.Get(t.placeholder))
			v.Del(t.placeholder)
		} else {
			seg = append(seg, t.pattern)
		}

		t = t.parent
	}

	sb := bytes.Buffer{}
	for i := len(seg) - 1; i >= 0; i-- {
		sb.WriteByte('/')
		sb.WriteString(seg[i])
	}

	return sb.Bytes()
}

func (p *_route) For(method string, h HandlerFunc) Route {
	p.hmap[method] = h.Connect(p.cc...)
	return p
}

func (p *_route) Get(h HandlerFunc) Route {
	return p.For(http.MethodGet, h)
}

func (p *_route) Post(h HandlerFunc) Route {
	return p.For(http.MethodPost, h.Connect(p.cc...))
}

func (p *_route) Put(h HandlerFunc) Route {
	return p.For(http.MethodPut, h.Connect(p.cc...))
}

func (p *_route) Delete(h HandlerFunc) Route {
	return p.For(http.MethodDelete, h.Connect(p.cc...))
}

func (p *_route) Patch(h HandlerFunc) Route {
	return p.For(http.MethodPatch, h.Connect(p.cc...))
}

func (p *_route) Options(h HandlerFunc) Route {
	return p.For(http.MethodOptions, h.Connect(p.cc...))
}

func (p *_route) Head(h HandlerFunc) Route {
	return p.For(http.MethodHead, h.Connect(p.cc...))
}

func (p *_route) Any(h HandlerFunc) Route {
	return p.For(string(wildcard), h.Connect(p.cc...))
}
