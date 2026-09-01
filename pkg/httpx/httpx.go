// Package httpx holds the Echo wiring that keeps the HTTP surface of this API
// exactly as it has always been: the JSON encoding of responses, the plain text
// "404 page not found" body used for both unknown routes and unknown methods,
// the empty 500 produced by a panic, the trailing slash redirect performed
// before any middleware runs, and the static file mount.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/pkg/binding"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// contentTypeJSON is the value written for every JSON response body.
const contentTypeJSON = "application/json; charset=utf-8"

// default404Body is the body served for an unmatched route.
var default404Body = []byte("404 page not found")

// NewServer builds the HTTP server with the same observable surface the API has
// always had: a request logger, panic recovery that answers with a bodiless
// 500, the plain text 404 used for unknown routes and unknown methods, the
// trailing slash redirect performed ahead of every middleware, and the request
// binding and JSON encoding rules the handlers depend on.
func NewServer() *echo.Echo {
	server := echo.New()

	server.Binder = &binding.Binder{}
	server.JSONSerializer = JSONSerializer{}
	server.HTTPErrorHandler = ErrorHandler

	server.Pre(RedirectTrailingSlash(server))
	server.Use(middleware.Logger())
	server.Use(Recover())

	return server
}

// JSONSerializer writes compact JSON without a trailing newline and labels it
// with a lowercase charset. Echo's default serializer streams through
// json.Encoder, which appends a newline, and tags responses as
// "application/json" only.
type JSONSerializer struct{}

var _ echo.JSONSerializer = (*JSONSerializer)(nil)

// Serialize encodes i onto the response.
func (JSONSerializer) Serialize(c echo.Context, i any, indent string) error {
	var (
		b   []byte
		err error
	)
	if indent != "" {
		b, err = json.MarshalIndent(i, "", indent)
	} else {
		b, err = json.Marshal(i)
	}
	if err != nil {
		return err
	}

	c.Response().Header().Set(echo.HeaderContentType, contentTypeJSON)
	_, err = c.Response().Write(b)
	return err
}

// Deserialize decodes the request body onto i.
func (JSONSerializer) Deserialize(c echo.Context, i any) error {
	return json.NewDecoder(c.Request().Body).Decode(i)
}

// ErrorHandler renders an unmatched route and an unmatched method the same way:
// a plain text 404. Every other error becomes a bodiless response carrying the
// status of the error.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	res := c.Response()
	res.Header().Del(echo.HeaderAllow)

	if code == http.StatusNotFound || code == http.StatusMethodNotAllowed {
		res.Header().Set(echo.HeaderContentType, "text/plain")
		res.WriteHeader(http.StatusNotFound)
		if _, werr := res.Write(default404Body); werr != nil {
			log.Println(werr)
		}
		return
	}

	res.WriteHeader(code)
}

// Recover turns a panic into a bodiless 500 and prints the stack trace.
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[Recovery] panic recovered:\n%v\n%s", r, debug.Stack())
					if !c.Response().Committed {
						c.Response().WriteHeader(http.StatusInternalServerError)
					}
					err = nil
				}
			}()
			return next(c)
		}
	}
}

var (
	regSafePrefix         = regexp.MustCompile("[^a-zA-Z0-9/-]+")
	regRemoveRepeatedChar = regexp.MustCompile("/{2,}")
)

// RedirectTrailingSlash must be registered with Echo.Pre so that it runs before
// the router and before any middleware. A request whose path only differs from
// a registered route by a trailing slash is answered with a redirect and never
// reaches the middleware chain; anything else is passed straight through.
//
// Clearing URL.RawPath additionally makes the router match on the decoded path,
// so a percent encoded separator inside a path segment does not silently become
// a route match.
func RedirectTrailingSlash(e *echo.Echo) echo.MiddlewareFunc {
	notFound := reflect.ValueOf(echo.NotFoundHandler).Pointer()
	methodNotAllowed := reflect.ValueOf(echo.MethodNotAllowedHandler).Pointer()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req.URL.RawPath = ""

			rPath := req.URL.Path
			if req.Method == http.MethodConnect || rPath == "/" {
				return next(c)
			}

			routed := func(p string) bool {
				if p == "" {
					return false
				}
				probe := e.NewContext(req, nil)
				e.Router().Find(req.Method, p, probe)
				handler := reflect.ValueOf(probe.Handler()).Pointer()
				return handler != notFound && handler != methodNotAllowed
			}

			if routed(rPath) {
				return next(c)
			}

			// A trailing slash redirect applies in either direction.
			var candidate string
			if strings.HasSuffix(rPath, "/") {
				candidate = strings.TrimSuffix(rPath, "/")
			} else {
				candidate = rPath + "/"
			}
			if !routed(candidate) {
				return next(c)
			}

			p := rPath
			if prefix := path.Clean(req.Header.Get("X-Forwarded-Prefix")); prefix != "." {
				prefix = regSafePrefix.ReplaceAllString(prefix, "")
				prefix = regRemoveRepeatedChar.ReplaceAllString(prefix, "/")
				p = prefix + "/" + rPath
			}

			req.URL.Path = p + "/"
			if length := len(p); length > 1 && p[length-1] == '/' {
				req.URL.Path = p[:length-1]
			}

			code := http.StatusMovedPermanently
			if req.Method != http.MethodGet {
				code = http.StatusTemporaryRedirect
			}

			http.Redirect(c.Response(), req, req.URL.String(), code)
			return nil
		}
	}
}

// SingleSegmentParams rejects a match in which a named path parameter spans a
// path separator. The router lets a trailing named parameter swallow the rest
// of the path, so "/api/user/a/b" would otherwise reach the "/api/user/:id"
// handler instead of being unmatched. Catch-all parameters are left alone.
func SingleSegmentParams() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for i, name := range c.ParamNames() {
				if name == "*" {
					continue
				}
				if strings.Contains(c.ParamValues()[i], "/") {
					return echo.ErrNotFound
				}
			}
			return next(c)
		}
	}
}

// Static mounts root under prefix for GET and HEAD. A missing file yields a
// bodiless 404 rather than the plain text one used for unmatched routes.
func Static(e *echo.Echo, prefix, root string) {
	fs := http.Dir(root)
	fileServer := http.StripPrefix(prefix, http.FileServer(fs))

	handler := func(c echo.Context) error {
		f, err := fs.Open("/" + c.Param("*"))
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		if cerr := f.Close(); cerr != nil {
			log.Println(cerr)
		}

		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	}

	e.GET(prefix+"/*", handler)
	e.HEAD(prefix+"/*", handler)
}
