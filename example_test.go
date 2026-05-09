package groxy_test

import (
	"bytes"
	"log"

	"github.com/SalzDevs/groxy"
)

func ExampleNew() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = proxy
}

func ExampleProxy_Use() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := proxy.Use(
		groxy.AddRequestHeader("X-Groxy-Request", "true"),
		groxy.AddResponseHeader("X-Groxy-Response", "true"),
	); err != nil {
		log.Fatal(err)
	}
}

func ExampleProxy_OnRequest() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
		ctx.Request.Header.Set("X-From-Groxy", "true")
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}

func ExampleBlock() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
		if ctx.Request.URL.Hostname() == "blocked.example" {
			return groxy.Block(403, "blocked by policy")
		}

		return nil
	}); err != nil {
		log.Fatal(err)
	}
}

func ExampleTransformRequestBody() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := proxy.Use(groxy.TransformRequestBody(func(body []byte) ([]byte, error) {
		return bytes.ReplaceAll(body, []byte("secret"), []byte("[redacted]")), nil
	})); err != nil {
		log.Fatal(err)
	}
}
