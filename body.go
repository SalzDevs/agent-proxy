package groxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// BodyTooLargeError is returned when a body helper reads more than the
// configured maximum body size.
type BodyTooLargeError struct {
	Limit int64
}

func (e *BodyTooLargeError) Error() string {
	return fmt.Sprintf("body exceeds maximum size of %d bytes", e.Limit)
}

// BodyBytes reads and restores the request body.
//
// HTTP bodies are streams: reading them consumes them. BodyBytes puts the bytes
// back with SetBody so later hooks and the proxy forwarding logic can read the
// body again.
func (ctx *RequestContext) BodyBytes() ([]byte, error) {
	if ctx.Request == nil || ctx.Request.Body == nil {
		return nil, nil
	}

	body, err := readBodyWithLimit(ctx.Request.Body, ctx.maxBodySize)
	if err != nil {
		return nil, err
	}

	ctx.SetBody(body)
	return body, nil
}

// SetBody replaces the request body with body.
func (ctx *RequestContext) SetBody(body []byte) {
	if ctx.Request == nil {
		return
	}

	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
	ctx.Request.ContentLength = int64(len(body))
	ctx.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	setContentLength(ctx.Request.Header, len(body))
}

// BodyBytes reads and restores the response body.
//
// HTTP bodies are streams: reading them consumes them. BodyBytes puts the bytes
// back with SetBody so later hooks and the response writing logic can read the
// body again.
func (ctx *ResponseContext) BodyBytes() ([]byte, error) {
	if ctx.Response == nil || ctx.Response.Body == nil {
		return nil, nil
	}

	body, err := readBodyWithLimit(ctx.Response.Body, ctx.maxBodySize)
	if err != nil {
		return nil, err
	}

	ctx.SetBody(body)
	return body, nil
}

// SetBody replaces the response body with body.
func (ctx *ResponseContext) SetBody(body []byte) {
	if ctx.Response == nil {
		return
	}

	ctx.Response.Body = io.NopCloser(bytes.NewReader(body))
	ctx.Response.ContentLength = int64(len(body))
	setContentLength(ctx.Response.Header, len(body))
}

func readBodyWithLimit(body io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(body)
	}

	limited := io.LimitReader(body, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, &BodyTooLargeError{Limit: limit}
	}

	return data, nil
}

func setContentLength(header http.Header, length int) {
	if header == nil {
		return
	}

	header.Set("Content-Length", strconv.Itoa(length))
}
