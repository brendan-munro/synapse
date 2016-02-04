package synapse

import (
	"net/http"

	"golang.org/x/net/context"
)

// HandlerFunc type is an adapter to allow the use of ordinary functions as
// handlers.  If fn is a function with the appropriate signature, HandlerFunc(f)
// is a Handler object that calls fn
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// ServeHTTP calls fn.ServeHTTPC(context.Background(), w, r)
func (fn HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.ServeHTTPC(context.Background(), w, r)
}

// ServeHTTPC calls fn(c, w, r)
func (fn HandlerFunc) ServeHTTPC(c context.Context, w http.ResponseWriter, r *http.Request) {
	fn(c, w, r)
}

// Handler interface can be implemented on a object to serve a particular path or
// subtree in the HTTP server.
//
// ServerHTTPC should write reply headers and data to the http.ResponseWriter and
// then return.  Return signals that the request is finished and that the HTTP server
// can move on to the next request on the connection.  Additionally, information can
// be shared using context.Context within a given request.
//
// ServerHTTP should write reply headers and data to the http.ResponseWriter and
// then return.  Return signals that the request is finished and that the HTTP server
// can move on to the next request on the connection.
type Handler interface {
	ServeHTTPC(c context.Context, w http.ResponseWriter, r *http.Request)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// A Constructor for a piece of middleware.
type Constructor func(next Handler) Handler

// Chain acts as a list of Hander constructors.  It is effectively immutable once
// created; as it will always hold the same set of constructors in the same order.
type Chain struct {
	constructors []Constructor
}

// NewChain creates a new Chain memorizing the given list of middlware Constructors
func NewChain(constructors ...Constructor) (c Chain) {
	c.constructors = append(c.constructors, constructors...)
	return c
}

// Then chains the middleware and returns the final Handler.
//
// A chain can be safely reused calling Then() serveral times.  Note that constructors
// are called on every call to Then() and thus serveral instances of the same middleware
// will be created what a Chain is reused in this way.
//
// Nil with result in a panic.
func (c Chain) Then(h Handler) Handler {
	if h == nil {
		panic("expected HandlerFunc")
	}

	var final = h

	for i := len(c.constructors) - 1; i >= 0; i-- {
		final = c.constructors[i](final)
	}

	return final
}

// ThenFunc works identically to Then but takes a HandlerFunc instead of a Handler.
func (c Chain) ThenFunc(fn HandlerFunc) Handler {
	return c.Then(HandlerFunc(fn))
}

// Append extends a Chain adding the specified constructors as the last ones in the
// request flow.
//
// Append returns a new chain leaving the original untoched.
func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, len(c.constructors)+len(constructors))
	copy(newCons, c.constructors)
	copy(newCons[len(c.constructors):], constructors)

	newChain := NewChain(newCons...)
	return newChain
}

// Extend extends a Chain by adding the specified Chain as the last on in the
// request flow.
//
// Extend returns a new Chain leaving the original untouched.
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.constructors...)
}
