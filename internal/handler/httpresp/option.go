package httpresp

// ConfigKey returns the gin.Context key under which httpresp.JSON stores
// the Config for downstream infrastructure (e.g. the HTTP cache middleware).
func ConfigKey() string {
	return configKey
}

// Option configures the HTTP representation metadata for a single JSON
// response. Options are pure value descriptors — they describe facts about the
// representation (e.g. which validator to use) and must never execute HTTP
// behaviour such as writing headers, reading request headers, or short-circuiting
// the response.
type Option func(*Config)

// Config carries HTTP representation metadata attached to a single response.
// Downstream infrastructure (e.g. the HTTP cache middleware) reads Config from
// the gin.Context after the handler has called JSON.
type Config struct {
	Metadata Metadata
}

// Metadata holds read-only descriptors for the HTTP representation. Each field
// is a factual statement about the representation, not an instruction.
type Metadata struct {
	// Validator declares the representation validator for this response.
	// Concrete implementations live in httpcache; httpresp only knows the
	// interface so it can store the value without importing httpcache.
	Validator Validator
}

// Validator is the abstract representation validator interface. A validator
// produces an ETag string for the current representation. Concrete
// implementations (Strong, Weak) live in the httpcache package; httpresp
// references only this interface to avoid a circular dependency.
type Validator interface {
	ETag() string
}
