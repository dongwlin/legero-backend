package httpresp

// ConfigKey returns the gin.Context key under which httpresp.JSON stores
// representation metadata for downstream infrastructure.
func ConfigKey() string {
	return configKey
}

// Option configures the HTTP representation metadata for a single JSON
// response. Options are pure value descriptors and must never execute HTTP
// behavior such as writing headers, reading request headers, or short-circuiting
// the response.
type Option func(*Config)

// Config carries HTTP representation metadata attached to a single response.
// Downstream infrastructure reads Config from the gin.Context after the
// handler has called JSON.
type Config struct {
	Metadata Metadata
}

// Metadata holds read-only descriptors for the HTTP representation. Each field
// is a factual statement about the representation, not an instruction.
type Metadata struct {
	// Validator carries an opaque representation-validator descriptor. The
	// infrastructure that owns the descriptor is responsible for interpreting
	// it; this package only stores and forwards the value.
	Validator any
}
