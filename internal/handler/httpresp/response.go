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
// stable strong ETag and support If-None-Match revalidation, without applying
// an endpoint-specific cache policy.
func JSON(c *gin.Context, status int, payload any) {
	renderJSON(c, status, payload, true, false)
}

// PrivateJSON renders an authenticated representation with a private,
// revalidation-required cache policy. It deliberately does not generate an
// ETag; use PrivateJSONWithETag for representations whose bytes are stable
// enough to revalidate. The policy is applied only to successful GET and HEAD
// representations.
func PrivateJSON(c *gin.Context, status int, payload any) {
	renderJSON(c, status, payload, false, true)
}

// PrivateJSONWithETag renders an authenticated GET or HEAD representation
// with a stable strong ETag and supports If-None-Match revalidation. The ETag
// is calculated from the exact bytes written to the response, so it remains
// correct for query-dependent and nested payloads.
func PrivateJSONWithETag(c *gin.Context, status int, payload any) {
	renderJSON(c, status, payload, true, true)
}

func renderJSON(c *gin.Context, status int, payload any, withETag, private bool) {
	body, err := ginjson.API.Marshal(payload)
	if err != nil {
		c.JSON(status, payload)
		return
	}

	eligible := etagMethod(c.Request) && etagStatus(status)
	if eligible && private {
		setPrivateCachePolicy(c)
	}

	if eligible && withETag {
		etag := strongETag(body)
		c.Header("ETag", etag)

		if matchesIfNoneMatch(strings.Join(c.Request.Header.Values("If-None-Match"), ","), etag) {
			c.Writer.Header().Del("Content-Length")
			c.Writer.Header().Del("Content-Type")
			c.AbortWithStatus(http.StatusNotModified)
			return
		}
	}

	if c.Request != nil && c.Request.Method == http.MethodHead {
		c.Header("Content-Type", jsonContentType)
		c.Header("Content-Length", strconv.Itoa(len(body)))
		c.Status(status)
		c.Writer.WriteHeaderNow()
		return
	}

	c.Data(status, jsonContentType, body)
}

func setPrivateCachePolicy(c *gin.Context) {
	c.Header("Cache-Control", "private, no-cache")
	appendVary(c.Writer.Header(), "Authorization")
}

func NoContent(c *gin.Context) {
	c.Status(204)
}
