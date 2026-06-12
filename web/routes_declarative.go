package web

// RouteBuilder represents a declaratively configured route.
type RouteBuilder struct {
	Method  string
	Pattern string
	Handler any
}

// DeclarativeGroup represents a prefix group of route definitions.
type DeclarativeGroup struct {
	Prefix string
	Routes []RouteBuilder
}

// GET declares a GET endpoint.
func GET(pattern string, handler any) RouteBuilder {
	return RouteBuilder{Method: "GET", Pattern: pattern, Handler: handler}
}

// POST declares a POST endpoint.
func POST(pattern string, handler any) RouteBuilder {
	return RouteBuilder{Method: "POST", Pattern: pattern, Handler: handler}
}

// PUT declares a PUT endpoint.
func PUT(pattern string, handler any) RouteBuilder {
	return RouteBuilder{Method: "PUT", Pattern: pattern, Handler: handler}
}

// DELETE declares a DELETE endpoint.
func DELETE(pattern string, handler any) RouteBuilder {
	return RouteBuilder{Method: "DELETE", Pattern: pattern, Handler: handler}
}

// Routes wraps a collection of RouteBuilder instances with a prefix path.
func Routes(prefix string, routes ...RouteBuilder) DeclarativeGroup {
	return DeclarativeGroup{Prefix: prefix, Routes: routes}
}

// RouteRegistrar is implemented by component struct types that export routes.
type RouteRegistrar interface {
	Routes() DeclarativeGroup
}
