package httpcache

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
)

// Middleware returns a gin.HandlerFunc that owns the response lifecycle. It
// replaces the ResponseWriter with a buffering writer so the status and headers
// are not committed to the wire while the handler runs; once the handler
// completes, the middleware generates the ETag (SHA-256 of the captured body
// for Strong validators, or the Validator's own string for Weak validators),
// checks If-None-Match, and sets the cache headers. Only then is the response
// committed.
//
// This deferred commit is what guarantees a HEAD response carries the same
// headers as the corresponding GET: writing the body (GET) or calling
// WriteHeaderNow (HEAD) inside the handler would freeze the header set before
// the ETag and cache headers exist.
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
		buf := &bufferingWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = buf

		defer func() {
			// Restore the original writer so outer handlers that write after
			// this middleware (e.g. the panic Recovery) go straight to the
			// client instead of into the buffer. On the panic path Commit is
			// never called and the partial buffer is discarded; body-describing
			// headers the aborted handler left behind (e.g. Content-Length set
			// by httpresp render) are removed so a Recovery 500 is not sent
			// with a length that no body will satisfy.
			c.Writer.Header().Del("Content-Length")
			c.Writer.Header().Del("Content-Type")
			c.Writer = buf.ResponseWriter
		}()

		c.Next()

		status := buf.Status()
		method := c.Request.Method

		var etag string
		notModified := false

		raw, exists := c.Get(httpresp.ConfigKey())
		if exists {
			if cfg, ok := raw.(*httpresp.Config); ok && cfg != nil && cfg.Metadata.Validator != nil {
				validator := cfg.Metadata.Validator
				if IsCacheableMethod(method) && IsCacheableStatus(status) {
					SetPrivateCachePolicy(c.Writer)

					// Weak validators produce the ETag string directly.
					// Strong validators signal via an empty ETag; the middleware
					// computes the hash from the captured body bytes.
					if v, ok := validator.(weakValidator); ok {
						etag = v.etag
					} else {
						// Strong: compute from captured body. For GET the body
						// was buffered by bufferingWriter.Write; for HEAD no
						// body was written, so we retrieve the marshaled bytes
						// stored by httpresp.JSON.
						body := buf.body.Bytes()
						if len(body) == 0 {
							body = httpresp.BodyFromContext(c)
						}
						etag = StrongETag(body)
					}
					if etag != "" {
						SetETag(c.Writer, etag)

						ifNoneMatch := c.GetHeader("If-None-Match")
						if ifNoneMatch == "" {
							ifNoneMatch = strings.Join(c.Request.Header.Values("If-None-Match"), ",")
						}
						if ifNoneMatch != "" && MatchIfNoneMatch(ifNoneMatch, etag) {
							notModified = true
						}
					}
				}
			}
		}

		buf.Commit(method, notModified)
	}
}

// bufferingWriter captures the response body and defers the header commit so
// the middleware can set the ETag and cache headers after the handler has
// completed. Without buffering, the header set would be frozen by the first
// body write (GET) or an explicit WriteHeaderNow (HEAD) before the middleware
// got a chance to add its headers.
type bufferingWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *bufferingWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *bufferingWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

// WriteHeader records the status code for Commit instead of forwarding it to
// the embedded writer. The header set is not committed here — that happens only
// in Commit, after the middleware has set the ETag and cache headers — so an
// explicit WriteHeader (c.Status, c.Writer.WriteHeader) inside a handler cannot
// freeze the headers before they exist.
func (w *bufferingWriter) WriteHeader(code int) {
	if code > 0 {
		w.status = code
	}
}

// Status returns the status recorded by WriteHeader, defaulting to 200 OK when
// the handler never set one explicitly.
func (w *bufferingWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// WriteHeaderNow is a no-op: the middleware commits the header after it has
// finished setting the ETag and cache headers.
func (w *bufferingWriter) WriteHeaderNow() {}

// Flush is a no-op while the response is buffered; streaming is not supported
// for responses flowing through the ETag middleware.
func (w *bufferingWriter) Flush() {}

func (w *bufferingWriter) Written() bool {
	return w.body.Len() > 0 || w.ResponseWriter.Written()
}

func (w *bufferingWriter) Size() int {
	return w.body.Len()
}

// Commit writes the status and buffered body to the underlying writer.
// notModified discards the body and representation headers per 304 semantics;
// for HEAD the headers are committed without writing a body.
func (w *bufferingWriter) Commit(method string, notModified bool) {
	if notModified {
		w.ResponseWriter.Header().Del("Content-Length")
		w.ResponseWriter.Header().Del("Content-Type")
		w.ResponseWriter.WriteHeader(http.StatusNotModified)
		w.ResponseWriter.WriteHeaderNow()
		return
	}
	if method == http.MethodHead {
		w.ResponseWriter.WriteHeader(w.Status())
		w.ResponseWriter.WriteHeaderNow()
		return
	}
	w.ResponseWriter.WriteHeader(w.Status())
	w.ResponseWriter.Write(w.body.Bytes())
}
