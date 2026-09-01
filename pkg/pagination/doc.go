// Package pagination is an in-repository port of github.com/Caknoooo/go-pagination
// v0.1.0 (MIT). Only the subset consumed by this project is kept, and the request
// binding helpers take an echo.Context instead of the upstream router context.
//
// The upstream module depends on the previous web framework, so it could not be
// kept as an external dependency without that framework surviving in go.mod and
// go.sum. Behaviour is byte-for-byte identical to the upstream implementation.
package pagination
