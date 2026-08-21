package httpcache

import "github.com/dongwlin/legero-backend/internal/handler/httpresp"

// WithValidator returns an httpresp.Option that attaches a Validator to the
// response metadata. The httpcache middleware reads this value from the
// gin.Context after the handler completes and uses it to generate an ETag,
// check If-None-Match, and set cache-related headers.
//
// Usage in a handler:
//
//	httpresp.JSON(c, http.StatusOK, resp, httpcache.WithValidator(httpcache.Weak("order", id, v)))
func WithValidator(v Validator) httpresp.Option {
	return func(cfg *httpresp.Config) {
		cfg.Metadata.Validator = v
	}
}
