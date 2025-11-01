package cache

import (
	"encoding/json"
)

type Serializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type jsonSerializer struct{}

func NewJSONSerializer() Serializer {
	return &jsonSerializer{}
}

func (s *jsonSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (s *jsonSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
