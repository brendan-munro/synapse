package synapse

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
)

func TestHandlerFunc(t *testing.T) {
	res := &httptest.ResponseRecorder{}
	req := &http.Request{}

	fn := HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		if want, have := context.Background(), c; want != have {
			t.Error("unexpected context:", have)
		}
		if want, have := res, w; want != have {
			t.Error("unexpected resposne:", have)
		}
		if want, have := req, r; want != have {
			t.Error("unexpected request:", have)
		}
	})

	fn.ServeHTTPC(context.Background(), res, req)
}

func TestHandlerFunc_SatisfiesHttpHandler(t *testing.T) {
	res := &httptest.ResponseRecorder{}
	req := &http.Request{}

	fn := HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		if want, have := context.Background(), c; want != have {
			t.Error("unexpected context:", have)
		}
		if want, have := res, w; want != have {
			t.Error("unexpected resposne:", have)
		}
		if want, have := req, r; want != have {
			t.Error("unexpected request:", have)
		}
	})

	fn.ServeHTTP(res, req)
}

func testLogMiddleware(logs io.Writer) Constructor {
	return func(next Handler) Handler {
		return HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
			logger := log.New(logs, "", 0)

			defer func(begin time.Time) {
				logger.Printf("method=%s route=%s took=%s", r.Method, r.RequestURI, time.Now().Sub(begin))
			}(time.Now())

			next.ServeHTTPC(c, w, r)
		})
	}
}

func testUserMiddleware() Constructor {
	return func(next Handler) Handler {
		return HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
			next.ServeHTTPC(context.WithValue(c, "user", "foobarbaz"), w, r)
		})
	}
}

func testContentTypeMiddleware() Constructor {
	return func(next Handler) Handler {
		return HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			next.ServeHTTPC(c, w, r)
		})
	}
}

func testHelloHandler() HandlerFunc {
	return HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!\n", c.Value("user"))
	})
}

func TestNewChain(t *testing.T) {
	c1 := func(Handler) Handler {
		return nil
	}
	c2 := func(Handler) Handler {
		return nil
	}

	slice := []Constructor{c1, c2}
	chain := NewChain(slice...)

	if want, have := true, funcsEqual(chain.constructors[0], c1); want != have {
		t.Error("unexpected constructor [0]:", have)
	}
	if want, have := true, funcsEqual(chain.constructors[1], c2); want != have {
		t.Error("unexpected constructor [1]:", have)
	}
}

func funcsEqual(f1, f2 interface{}) bool {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	return val1.Pointer() == val2.Pointer()
}

func TestChain_ThenPanicsIfHandlerNil(t *testing.T) {
	defer func() {
		recover()
	}()

	chain := NewChain().Then(nil)
	chain.ServeHTTPC(nil, nil, nil)

	t.Error("should have paniced")
}

func TestChain_ThenFuncPanicsIfHandlerNil(t *testing.T) {
	defer func() {
		recover()
	}()

	chain := NewChain().ThenFunc(nil)
	chain.ServeHTTPC(nil, nil, nil)

	t.Error("should have paniced")
}

func TestChain_ThenWorksWithNoMiddleware(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal("unexpected panic:", r)
		}
	}()

	chain := NewChain()
	final := chain.Then(testHelloHandler())

	if want, have := true, funcsEqual(final, testHelloHandler()); want != have {
		t.Error("unexpected final:", have)
	}
}

func TestChain_ThenFuncWorksWithNoMiddleware(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal("unexpected panic:", r)
		}
	}()

	chain := NewChain()
	final := chain.ThenFunc(testHelloHandler())

	if want, have := true, funcsEqual(final, testHelloHandler()); want != have {
		t.Error("unexpected final:", have)
	}
}

func TestChain_ThenOrdersHandlerFuncsRight(t *testing.T) {
	logs := &bytes.Buffer{}

	chained := NewChain(
		testLogMiddleware(logs),
		testUserMiddleware(),
		testContentTypeMiddleware(),
	).Then(testHelloHandler())

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTPC(context.Background(), w, r)

	if want, have := "text/plain", w.Header().Get("Content-Type"); want != have {
		t.Error("unexpected response header [Content-Type]:", have)
	}
	if want, have := "Hello, foobarbaz!\n", w.Body.String(); want != have {
		t.Error("unexpected response body:", have)
	}
}

func TestChain_AppendAddsHandlerFuncsCorrectly(t *testing.T) {
	logs := &bytes.Buffer{}

	chain := NewChain(testLogMiddleware(logs))
	newChain := chain.Append(testUserMiddleware(), testContentTypeMiddleware())

	if want, have := 1, len(chain.constructors); want != have {
		t.Error("unexpected len:", have)
	}
	if want, have := 3, len(newChain.constructors); want != have {
		t.Error("unexpected len:", have)
	}

	chained := newChain.Then(testHelloHandler())

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTPC(context.Background(), w, r)

	if want, have := "text/plain", w.Header().Get("Content-Type"); want != have {
		t.Error("unexpected response header [Content-Type]:", have)
	}
	if want, have := "Hello, foobarbaz!\n", w.Body.String(); want != have {
		t.Error("unexpected response body:", have)
	}
}

func TestChain_AppendRespectsImmutability(t *testing.T) {
	chain := NewChain(testUserMiddleware())
	newChain := chain.Append(testUserMiddleware())

	if one, two := &chain.constructors[0], &newChain.constructors[0]; one == two {
		t.Error("unexpected equality")
	}
}

func TestChain_ExtendAddsHandlerFuncsCorrectly(t *testing.T) {
	logs := &bytes.Buffer{}

	chain1 := NewChain(testLogMiddleware(logs))
	chain2 := NewChain(testUserMiddleware(), testContentTypeMiddleware())
	newChain := chain1.Extend(chain2)

	if want, have := 1, len(chain1.constructors); want != have {
		t.Error("unexpected len:", have)
	}
	if want, have := 2, len(chain2.constructors); want != have {
		t.Error("unexpected len:", have)
	}
	if want, have := 3, len(newChain.constructors); want != have {
		t.Error("unexpected len:", have)
	}

	chained := newChain.Then(testHelloHandler())

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTPC(context.Background(), w, r)

	if want, have := "text/plain", w.Header().Get("Content-Type"); want != have {
		t.Error("unexpected response header [Content-Type]:", have)
	}
	if want, have := "Hello, foobarbaz!\n", w.Body.String(); want != have {
		t.Error("unexpected response body:", have)
	}
}

func TestExtendRespectsImmutability(t *testing.T) {
	chain := NewChain(testUserMiddleware())
	newChain := chain.Extend(NewChain(testUserMiddleware()))

	if one, two := &chain.constructors[0], &newChain.constructors[0]; one == two {
		t.Error("unexpected equality")
	}

	assert.NotEqual(t, &chain.constructors[0], &newChain.constructors[0])
}
