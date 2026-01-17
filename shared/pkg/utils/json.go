package utils

import "encoding/json"

func MarshalJSONSafe(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func UnmarshalJSONSafe(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func MarshalRawMessageSafe(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return json.RawMessage(data)
}

func UnmarshalRawMessageSafe(data json.RawMessage, v interface{}) error {
	return json.Unmarshal(data, v)
}
