package binding

import "github.com/labstack/echo/v4"

// Binder plugs this package into Echo as the application wide request binder,
// so that echo.Context.Bind keeps the content-type dispatch, decoder errors and
// `binding` tag validation the API has always exposed.
type Binder struct{}

var _ echo.Binder = (*Binder)(nil)

// Bind decodes the request body onto i.
func (Binder) Bind(i any, c echo.Context) error {
	req := c.Request()
	b := Default(req.Method, FilterFlags(req.Header.Get("Content-Type")))
	return b.Bind(req, i)
}

// BindQuery decodes only the URL query string onto obj.
func BindQuery(c echo.Context, obj any) error {
	return Query.Bind(c.Request(), obj)
}
