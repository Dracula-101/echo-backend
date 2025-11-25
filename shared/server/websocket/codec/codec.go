package codec

import (
	"encoding/json"
	"fmt"
)

// Codec encodes and decodes messages
type Codec interface {
	Encode(v interface{}) ([]byte, error)
	Decode(data []byte, v interface{}) error
	Name() string
}

// JSONCodec is a JSON codec
type JSONCodec struct{}

// NewJSONCodec creates a new JSON codec
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}

// Encode encodes a value to JSON
func (c *JSONCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Decode decodes JSON data
func (c *JSONCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Name returns the codec name
func (c *JSONCodec) Name() string {
	return "json"
}

// Registry manages codecs
type Registry struct {
	codecs map[string]Codec
}

// NewRegistry creates a new codec registry
func NewRegistry() *Registry {
	return &Registry{
		codecs: make(map[string]Codec),
	}
}

// Register registers a codec
func (r *Registry) Register(codec Codec) {
	r.codecs[codec.Name()] = codec
}

// Get gets a codec by name
func (r *Registry) Get(name string) (Codec, error) {
	codec, exists := r.codecs[name]
	if !exists {
		return nil, fmt.Errorf("codec not found: %s", name)
	}
	return codec, nil
}
