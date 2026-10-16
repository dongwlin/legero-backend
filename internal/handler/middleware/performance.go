package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/infra/performance"
)

const PerformanceRequestIDHeader = "X-Performance-Request-ID"

// The client generates a fresh opaque session ID and monotonic request counter.
// Never accept arbitrary strings that could carry payloads or header syntax.
var performanceRequestIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|local-[a-z0-9]{8,32}):[1-9][0-9]{0,8}$`)

// PerformanceTiming enables timing only for explicitly correlated HTTP requests.
// Ordinary traffic uses the original writer and context with no trace allocation.
func PerformanceTiming() gin.HandlerFunc {
	return performanceTiming(time.Now)
}

func performanceTiming(now func() time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		ids := c.Request.Header.Values(PerformanceRequestIDHeader)
		if len(ids) != 1 || !performanceRequestIDPattern.MatchString(ids[0]) ||
			c.Request.Header.Get("Upgrade") != "" {
			c.Next()
			return
		}
		ctx, trace := performance.Start(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		writer := &performanceWriter{
			ResponseWriter: c.Writer,
			trace:          trace,
			startedAt:      now(),
			now:            now,
			requestID:      ids[0],
		}
		c.Writer = writer
		c.Next()
		writer.finish()
	}
}

type performanceWriter struct {
	gin.ResponseWriter
	trace     *performance.Trace
	startedAt time.Time
	now       func() time.Time
	requestID string
	finished  bool
}

func (w *performanceWriter) finish() {
	if w.finished {
		return
	}
	w.finished = true
	snapshot := w.trace.Finish()
	elapsed := max(w.now().Sub(w.startedAt), 0)
	w.Header().Set(PerformanceRequestIDHeader, w.requestID)
	w.Header().Set("Server-Timing", fmt.Sprintf(
		`app;dur=%.3f, db;dur=%.3f;desc="queries:%d"`,
		float64(elapsed)/float64(time.Millisecond),
		float64(snapshot.DBTime)/float64(time.Millisecond),
		snapshot.Queries,
	))
}

func (w *performanceWriter) WriteHeaderNow() {
	w.finish()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *performanceWriter) Write(data []byte) (int, error) {
	w.finish()
	return w.ResponseWriter.Write(data)
}

func (w *performanceWriter) WriteString(data string) (int, error) {
	w.finish()
	return w.ResponseWriter.WriteString(data)
}

func (w *performanceWriter) Flush() {
	w.finish()
	w.ResponseWriter.Flush()
}

func (w *performanceWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
