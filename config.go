package entadmin

import "net/http"

// Middleware has the same shape as standard net/http middleware. Entries are
// applied in declaration order, with Middleware[0] as the outermost wrapper.
type Middleware func(http.Handler) http.Handler

// Config controls construction of the router-independent admin handler.
type Config struct {
	// Registry is normally returned by generated Ent glue, for example
	// ent.AdminRegistry(client).
	Registry Registry

	// BasePath is the URL path at which the handler is mounted. It may be
	// empty, "/", "/admin", or "/admin/"; construction normalizes it to a
	// leading-slash path without a trailing slash, except for "/".
	BasePath string

	// Middleware protects and decorates the complete admin handler. The
	// package does not create sessions, users, authentication, or RBAC.
	Middleware []Middleware
}
