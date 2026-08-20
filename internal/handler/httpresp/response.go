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

// PrivateJSONWithVersionETag renders a single versioned resource using a weak
// version validator. The payload is lazy so a matching If-None-Match request
// can return 304 without converting the resource to its response DTO or
// serializing its body.
//
// No current route uses this helper yet; it is provided for single-resource
// GET/HEAD handlers when they are introduced. Complex list and aggregate
// responses must continue to use PrivateJSONWithETag.
func PrivateJSONWithVersionETag(c *gin.Context, status int, payload func() any, resource, id string, version int64) {
	renderVersionJSON(c, status, payload, resource, id, version)
}

func renderVersionJSON(c *gin.Context, status int, payload func() any, resource, id string, version int64) {
	eligible := etagMethod(c.Request) && etagStatus(status)
	if !eligible {
		renderJSON(c, status, payload(), false, true)
		return
	}

	setPrivateCachePolicy(c)
	etag := VersionETag(resource, id, version)
	c.Header("ETag", etag)
	if matchesIfNoneMatch(strings.Join(c.Request.Header.Values("If-None-Match"), ","), etag) {
		writeNotModified(c)
		return
	}
	if c.Request.Method == http.MethodHead {
		// The version validator and representation media type are known without
		// constructing the DTO. Content-Length is intentionally omitted when
		// determining it would require marshaling and discarding the body.
		c.Header("Content-Type", jsonContentType)
		c.Status(status)
		c.Writer.WriteHeaderNow()
		return
	}

	payloadValue := payload()
	body, err := ginjson.API.Marshal(payloadValue)
	if err != nil {
		// The precomputed validator describes the intended JSON representation;
		// do not leave it attached if marshaling fails and Gin falls back to its
		// error-capable renderer.
		c.Writer.Header().Del("ETag")
		c.JSON(status, payloadValue)
		return
	}
	renderJSONBody(c, status, body)
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
			writeNotModified(c)
			return
		}
	}

	renderJSONBody(c, status, body)
}

func renderJSONBody(c *gin.Context, status int, body []byte) {
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

func writeNotModified(c *gin.Context) {
	// A 304 never carries a response body. Keep the validator and cache policy
	// headers already set above, but remove representation metadata that would
	// describe a body which is not present.
	c.Writer.Header().Del("Content-Length")
	c.Writer.Header().Del("Content-Type")
	c.AbortWithStatus(http.StatusNotModified)
}

func NoContent(c *gin.Context) {
	c.Status(204)
}
