# Proxy authentication

Groxy provides HTTP Basic proxy authentication middleware for forward proxies.

Proxy authentication uses the proxy-specific header:

```http
Proxy-Authorization: Basic ...
```

When credentials are missing or invalid, Groxy returns:

```http
407 Proxy Authentication Required
Proxy-Authenticate: Basic realm="Groxy"
```

## Basic usage

Use `ProxyBasicAuth` when you have one static username/password pair:

```go
password := os.Getenv("GROXY_PROXY_PASSWORD")
if password == "" {
	log.Fatal("set GROXY_PROXY_PASSWORD")
}

proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
if err != nil {
	log.Fatal(err)
}

if err := proxy.Use(groxy.ProxyBasicAuth("admin", password)); err != nil {
	log.Fatal(err)
}
```

Test it with curl:

```bash
curl -x http://admin:password@127.0.0.1:8080 http://example.com
curl -x http://admin:password@127.0.0.1:8080 https://example.com
```

`ProxyBasicAuth` protects both normal HTTP proxy requests and HTTPS CONNECT
tunnels.

## Custom validation

Use `ProxyBasicAuthFunc` when credentials come from your own config, database,
or identity system:

```go
if err := proxy.Use(groxy.ProxyBasicAuthFunc(func(username, password string) bool {
	return users.Verify(username, password)
})); err != nil {
	log.Fatal(err)
}
```

If the validator is nil or returns false, Groxy rejects the request with `407
Proxy Authentication Required`.

## Security notes

Basic authentication is not encryption. The username and password are only
base64-encoded inside the `Proxy-Authorization` header.

Use Basic proxy authentication only when the client-to-proxy connection is
otherwise protected or trusted, such as:

- localhost development
- private network
- VPN
- SSH tunnel
- TLS-terminating wrapper or load balancer
- another secure transport controlled by your application

Avoid hardcoding passwords in source code. Prefer environment variables, secret
stores, or application configuration.

## Credential handling

Groxy treats proxy credentials as proxy-only data:

- `ProxyBasicAuth` compares static credentials in constant time.
- `ProxyBasicAuthFunc` passes parsed username/password values to your validator.
- `Proxy-Authorization` is stripped before forwarding requests upstream.
- `AccessLog` logs method, host, status, and duration only; it does not log
  headers or credentials.

Middleware you write yourself is still privileged code. Avoid logging sensitive
headers such as `Proxy-Authorization`, `Authorization`, and `Cookie`.

## HTTPS inspection

For normal HTTPS CONNECT tunneling, proxy authentication is checked before the
CONNECT tunnel is opened.

When HTTPS inspection is enabled, Groxy authenticates the CONNECT request before
starting inspection. Requests parsed inside the inspected TLS connection are not
re-authenticated, because the CONNECT tunnel has already passed proxy auth.

## Runnable example

See [`examples/proxy-auth`](../examples/proxy-auth):

```bash
GROXY_PROXY_PASSWORD=password go run ./examples/proxy-auth
```

Then, from another terminal:

```bash
curl -x http://admin:password@127.0.0.1:8080 http://example.com
```
