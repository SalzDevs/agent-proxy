package groxy

import "net"

// BodyTransform transforms a request or response body.
type BodyTransform func([]byte) ([]byte, error)

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

// TransformRequestBody returns middleware that replaces a request body with the
// bytes returned by transform.
func TransformRequestBody(transform BodyTransform) Middleware {
	return OnRequest(func(ctx *RequestContext) error {
		body, err := ctx.BodyBytes()
		if err != nil {
			return err
		}

		updatedBody, err := transform(body)
		if err != nil {
			return err
		}

		ctx.SetBody(updatedBody)
		return nil
	})
}

// TransformResponseBody returns middleware that replaces a response body with
// the bytes returned by transform.
func TransformResponseBody(transform BodyTransform) Middleware {
	return OnResponse(func(ctx *ResponseContext) error {
		body, err := ctx.BodyBytes()
		if err != nil {
			return err
		}

		updatedBody, err := transform(body)
		if err != nil {
			return err
		}

		ctx.SetBody(updatedBody)
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
