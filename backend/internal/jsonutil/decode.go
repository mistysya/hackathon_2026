// Package jsonutil provides strict decoding for one JSON document.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

var ErrTrailingJSON = errors.New("unexpected trailing JSON value")

// DecodeStrict decodes exactly one JSON value into target and rejects unknown
// object fields and any trailing JSON value.
func DecodeStrict(input io.Reader, target any) error {
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return requireEOF(decoder)
}

// DecodeSingle decodes exactly one JSON value. It preserves JSON numbers for
// callers that need to validate them against a schema before decoding a type.
func DecodeSingle(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := requireEOF(decoder); err != nil {
		return nil, err
	}
	return value, nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	}
	return ErrTrailingJSON
}
