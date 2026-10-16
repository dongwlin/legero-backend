package httpresp

import "github.com/gin-gonic/gin"

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

// JSON renders a v2 JSON response.
func JSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}
