package kafka

import (
	"encoding/json"
)

type Serializer interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

type jsonSerializer struct{}

func NewJSONSerializer() Serializer {
	return &jsonSerializer{}
}

func (s *jsonSerializer) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (s *jsonSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
