# Custom block and error responses

Groxy hooks can stop requests with `groxy.Block(statusCode, message)`. This is
useful for policy blocks, allow/deny lists, and simple API-style errors.

## Custom text block

```go
if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	if ctx.Request.URL.Hostname() == "blocked.example" {
		return groxy.Block(http.StatusForbidden, "blocked by company policy")
	}
	return nil
}); err != nil {
	log.Fatal(err)
}
```

The same pattern works for CONNECT tunnels:

```go
if err := proxy.OnConnect(func(ctx *groxy.ConnectContext) error {
	if ctx.Host == "private.example:443" {
		return groxy.Block(http.StatusForbidden, "CONNECT blocked by company policy")
	}
	return nil
}); err != nil {
	log.Fatal(err)
}
```

## API-style block body

`Block` accepts any string body, so applications can return a structured body for
API clients:

```go
if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	if ctx.Request.URL.Hostname() == "169.254.169.254" {
		return groxy.Block(http.StatusForbidden, `{"error":"metadata_blocked","message":"metadata service is blocked"}`)
	}
	return nil
}); err != nil {
	log.Fatal(err)
}
```

Today, block responses are written with Go's `http.Error`, so the response uses
Go's default text error format and content type. If you need full control over
block response headers, content type, or serialization, track the future custom
error handler work in GitHub issues.

## Reformat upstream error responses

Response hooks can also turn upstream error responses into a consistent API
format before they reach the client:

```go
if err := proxy.OnResponse(func(ctx *groxy.ResponseContext) error {
	if ctx.Response.StatusCode < 500 {
		return nil
	}

	body := []byte(`{"error":"upstream_unavailable","message":"upstream returned an error"}` + "\n")

	ctx.Response.StatusCode = http.StatusBadGateway
	ctx.Response.Status = "502 Bad Gateway"
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.Header.Del("Content-Encoding")
	ctx.SetBody(body)

	return nil
}); err != nil {
	log.Fatal(err)
}
```

`SetBody` updates `Content-Length` for you.

## Runnable example

See [`examples/custom-block-response`](../examples/custom-block-response):

```bash
go run ./examples/custom-block-response
```
