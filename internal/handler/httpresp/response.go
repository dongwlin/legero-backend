package httpresp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ginjson "github.com/gin-gonic/gin/codec/json"
)

const (
	jsonContentType = "application/json; charset=utf-8"
	configKey       = "httpresp.config"
	bodyKey         = "httpresp.body"
)

// Response is the v2 unified envelope. All v2 API responses wrap their
// business payload inside this structure so clients can rely on a stable
// contract: code for machine-readable status, message for human-readable
// context, and data for the business payload.
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Success wraps data in the v2 envelope with a zero code and "success" message.
func Success(data any) Response {
	return Response{Code: "0", Message: "success", Data: data}
}

// JSON renders a v2 JSON response. It serializes body, sets Content-Type, and
// writes the response. For HEAD requests the body is not written; the status
// and headers are recorded and the header commit is left to the owning HTTP
// cache middleware, which must set ETag and cache headers before the wire
// header is finalized (a premature commit would freeze the header set). Any
// Option values (such as a cache validator declared by the handler) are stored
// in the gin.Context for downstream infrastructure (e.g. the HTTP cache
// middleware) to consume — httpresp itself never acts on them.
func JSON(c *gin.Context, status int, body any, opts ...Option) {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	data, err := ginjson.API.Marshal(body)
	if err != nil {
		c.JSON(status, body)
		return
	}

	c.Set(configKey, &cfg)
	c.Set(bodyKey, data)
	renderBody(c, status, data)
}

// NoContent sends a 204 with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BodyFromContext retrieves the marshaled response body bytes stored by JSON.
// Returns nil if no body was stored (e.g. handler did not call JSON, or
// marshaling failed).
func BodyFromContext(c *gin.Context) []byte {
	raw, exists := c.Get(bodyKey)
	if !exists {
		return nil
	}
	data, _ := raw.([]byte)
	return data
}

func renderBody(c *gin.Context, status int, body []byte) {
	if c.Request != nil && c.Request.Method == http.MethodHead {
		c.Header("Content-Type", jsonContentType)
		c.Header("Content-Length", itoa(len(body)))
		c.Status(status)
		return
	}
	c.Data(status, jsonContentType, body)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
