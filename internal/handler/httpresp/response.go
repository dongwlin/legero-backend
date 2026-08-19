package httpresp

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	ginjson "github.com/gin-gonic/gin/codec/json"
)

const jsonContentType = "application/json; charset=utf-8"

// JSON renders a JSON response. Successful GET and HEAD representations get a
// stable strong ETag and support If-None-Match revalidation. The ETag is
// calculated from the exact bytes written to the response, so it remains
// correct for query-dependent and nested payloads.
func JSON(c *gin.Context, status int, payload any) {
	if etagMethod(c.Request) && etagStatus(status) {
		body, err := ginjson.API.Marshal(payload)
		if err == nil {
			etag := strongETag(body)
			c.Header("ETag", etag)
			c.Header("Cache-Control", "private, no-cache")
			appendVary(c.Writer.Header(), "Authorization")

			if matchesIfNoneMatch(strings.Join(c.Request.Header.Values("If-None-Match"), ","), etag) {
				c.Writer.Header().Del("Content-Length")
				c.Writer.Header().Del("Content-Type")
				c.AbortWithStatus(http.StatusNotModified)
				return
			}

			if c.Request.Method == http.MethodHead {
				c.Header("Content-Type", jsonContentType)
				c.Header("Content-Length", strconv.Itoa(len(body)))
				c.Status(status)
				c.Writer.WriteHeaderNow()
				return
			}

			c.Data(status, jsonContentType, body)
			return
		}
	}

	c.JSON(status, payload)
}

func NoContent(c *gin.Context) {
	c.Status(204)
}
