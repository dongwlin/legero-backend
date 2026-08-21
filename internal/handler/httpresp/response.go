package httpresp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ginjson "github.com/gin-gonic/gin/codec/json"
)

const jsonContentType = "application/json; charset=utf-8"

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
// writes the response. Any Option values (such as a cache validator declared by
// the handler) are stored in the gin.Context for downstream infrastructure
// (e.g. httpcache middleware) to consume — httpresp itself never acts on them.
func JSON(c *gin.Context, status int, body any, opts ...Option) {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	c.Set(configKey, &cfg)

	data, err := ginjson.API.Marshal(body)
	if err != nil {
		c.JSON(status, body)
		return
	}
	renderBody(c, status, data)
}

// NoContent sends a 204 with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func renderBody(c *gin.Context, status int, body []byte) {
	if c.Request != nil && c.Request.Method == http.MethodHead {
		c.Header("Content-Type", jsonContentType)
		c.Header("Content-Length", itoa(len(body)))
		c.Status(status)
		c.Writer.WriteHeaderNow()
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
