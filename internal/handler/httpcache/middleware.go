package httpcache

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
)

// Middleware returns a gin.HandlerFunc that reads the response metadata stored
// by httpresp.JSON and, when a Validator is present, generates an ETag, checks
// If-None-Match, and sets cache headers. For Strong validators the middleware
// captures the response body to compute SHA-256; for Weak validators it uses
// the ETag string produced by the Validator directly.
//
// The middleware must run after the handler (i.e. after httpresp.JSON has
// written the body and stored Config in gin.Context). Register it with a
// route-group-level use() so it fires for every handler in that group.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
			// Strong: compute from captured body.
			if cw, ok := c.Writer.(*capturingWriter); ok {
				etag = StrongETag(cw.body.Bytes())
			}
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
			c.Writer.Header().Del("Content-Length")
			c.Writer.Header().Del("Content-Type")
			c.Writer.WriteHeader(http.StatusNotModified)
			c.Abort()
			return
		}

		// For HEAD responses the body has already been captured but should
		// not be written; httpresp.JSON already handled that.
		if method == http.MethodHead {
			c.Writer.Header().Del("Content-Length")
			c.Writer.Header().Del("Content-Type")
			c.Status(status)
			c.Writer.WriteHeaderNow()
		}
	}
}

// WrapWriter wraps the gin.ResponseWriter so the middleware can capture the
// response body for Strong ETag computation. Register this as a wrapper in
// the route group that uses the cache middleware.
func WrapWriter(c *gin.Context) {
	c.Writer = &capturingWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
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


