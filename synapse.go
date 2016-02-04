package synapse

import (
	"net/http"

	"golang.org/x/net/context"
)

type HandlerFunc func(c context.Context, w http.ResponseWriter, r *http.Request)

func (fn HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn(context.Background(), w, r)
}

type Constructor func(next HandlerFunc) HandlerFunc

type Chain []Constructor

func NewChain(constructors ...Constructor) Chain {
	c := Chain{}
	c = append(c, constructors...)

	return c
}

func (c Chain) Then(h HandlerFunc) HandlerFunc {
	if h == nil {
		panic("expected HandlerFunc")
	}

	var final = h

	for i := len(c) - 1; i >= 0; i-- {
		final = c[i](final)
	}

	return final
}

func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, len(c)+len(constructors))
	copy(newCons, c)
	copy(newCons[len(c):], constructors)

	newChain := NewChain(newCons...)
	return newChain
}

func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain...)
}
