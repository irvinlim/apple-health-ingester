package healthautoexport

import (
	"bytes"
	"encoding/json"
	"io"

	jsoniter "github.com/json-iterator/go"
)

// Marshal payload to io.Writer.
func Marshal(payload *Payload, w io.Writer) error {
	enc := jsoniter.NewEncoder(w)
	if err := enc.Encode(payload); err != nil {
		return err
	}
	return nil
}

// MarshalToString marshals payload to a string.
func MarshalToString(payload *Payload) (string, error) {
	buf := new(bytes.Buffer)
	if err := Marshal(payload, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Unmarshal payload from io.Reader.
func Unmarshal(r io.Reader) (*Payload, error) {
	var payload Payload
	dec := json.NewDecoder(r)
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// UnmarshalFromString unmarshals payload from a string.
func UnmarshalFromString(s string) (*Payload, error) {
	return Unmarshal(bytes.NewBufferString(s))
}
