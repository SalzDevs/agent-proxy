package groxy

import "net"

// AddRequestHeader returns middleware that sets a request header before the
// request is sent upstream.
func AddRequestHeader(key, value string) Middleware {
	return OnRequest(func(ctx *RequestContext) error {
		ctx.Request.Header.Set(key, value)
		return nil
	})
}

// AddResponseHeader returns middleware that sets a response header before the
// response is sent back to the client.
func AddResponseHeader(key, value string) Middleware {
	return OnResponse(func(ctx *ResponseContext) error {
		ctx.Response.Header.Set(key, value)
		return nil
	})
}

// RemoveRequestHeader returns middleware that deletes a request header before
// the request is sent upstream.
func RemoveRequestHeader(key string) Middleware {
	return OnRequest(func(ctx *RequestContext) error {
		ctx.Request.Header.Del(key)
		return nil
	})
}

// RemoveResponseHeader returns middleware that deletes a response header before
// the response is sent back to the client.
func RemoveResponseHeader(key string) Middleware {
	return OnResponse(func(ctx *ResponseContext) error {
		ctx.Response.Header.Del(key)
		return nil
	})
}

// BlockHost returns middleware that blocks normal HTTP requests to host.
func BlockHost(host string, statusCode int, message string) Middleware {
	return OnRequest(func(ctx *RequestContext) error {
		if ctx.Request.URL.Hostname() == host {
			return Block(statusCode, message)
		}

		return nil
	})
}

// BlockConnectHost returns middleware that blocks CONNECT tunnels to host.
func BlockConnectHost(host string, statusCode int, message string) Middleware {
	return OnConnect(func(ctx *ConnectContext) error {
		if connectHostname(ctx.Host) == host {
			return Block(statusCode, message)
		}

		return nil
	})
}

func connectHostname(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}

	return host
}
