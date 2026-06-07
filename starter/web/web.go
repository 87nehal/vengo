package webstarter

import "github.com/87nehal/vengo/web"

func Module(addr string) *web.Server {
	return web.New(addr)
}
