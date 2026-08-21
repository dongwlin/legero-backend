package httpcache

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
)

// Middleware returns a gin.HandlerFunc that wraps the ResponseWriter to capture
// the response body, then after the handler completes reads the response
// metadata stored by httpresp.JSON and, when a Validator is present, generates
// an ETag, checks If-None-Match, and sets cache headers.
//
// For Strong validators the middleware computes SHA-256 from the captured body
// bytes (GET) or by re-marshaling the response body (HEAD, since HEAD never
// writes a body). For Weak validators it uses the ETag string produced by the
// Validator directly.
//
// This middleware owns the writer lifecycle — callers must not register a
// separate WrapWriter middleware.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		capturing := &capturingWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = capturing

		c.Next()

		raw, exists := c.Get(httpresp.ConfigKey())
		if !exists {
			return
		}
		cfg, ok := raw.(*httpresp.Config)
		if !ok || cfg == nil || cfg.Metadata.Validator == nil {
			return
		}
		validator := cfg.Metadata.Validator

		status := c.Writer.Status()
		method := c.Request.Method
		if !IsCacheableMethod(method) || !IsCacheableStatus(status) {
			return
		}

		SetPrivateCachePolicy(c.Writer)

		// Weak validators produce the ETag string directly.
		// Strong validators signal via an empty ETag; the middleware computes
		// the hash from the captured body bytes.
		var etag string
		if v, ok := validator.(weakValidator); ok {
			etag = v.etag
		} else {
			// Strong: compute from captured body. For GET the body was
			// intercepted by capturingWriter.Write; for HEAD no body was
			// written, so we retrieve the marshaled bytes stored by
			// httpresp.JSON.
			body := capturing.body.Bytes()
			if len(body) == 0 {
				body = httpresp.BodyFromContext(c)
			}
			etag = StrongETag(body)
		}
		if etag == "" {
			return
		}

		SetETag(c.Writer, etag)

		ifNoneMatch := c.GetHeader("If-None-Match")
		if ifNoneMatch == "" {
			ifNoneMatch = strings.Join(c.Request.Header.Values("If-None-Match"), ",")
		}
		if ifNoneMatch != "" && MatchIfNoneMatch(ifNoneMatch, etag) {
			WriteNotModified(c.Writer)
			c.Abort()
			return
		}

		// For HEAD responses the status and headers have already been sent
		// by httpresp.JSON; only flush without writing a body.
		if method == http.MethodHead {
			c.Writer.Header().Del("Content-Length")
			c.Writer.Header().Del("Content-Type")
			if !c.Writer.Written() {
				c.Writer.WriteHeaderNow()
			}
		}
	}
}

// capturingWriter intercepts Write calls to capture the body for Strong ETag
// hashing while still forwarding bytes to the underlying ResponseWriter.
type capturingWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *capturingWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *capturingWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *capturingWriter) Status() int {
	return w.ResponseWriter.Status()
}

func (w *capturingWriter) Written() bool {
	return w.ResponseWriter.Written()
}

func (w *capturingWriter) WriteHeaderNow() {
	w.ResponseWriter.WriteHeaderNow()
}

func (w *capturingWriter) Pusher() http.Pusher {
	return w.ResponseWriter.Pusher()
}

func (w *capturingWriter) Flush() {
	w.ResponseWriter.Flush()
}

func (w *capturingWriter) Size() int {
	return w.ResponseWriter.Size()
}


