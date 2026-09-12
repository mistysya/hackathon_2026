package jsonutil

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeStrictRejectsUnknownAndTrailingValues(t *testing.T) {
	for name, input := range map[string]string{
		"unknown field":  `{"name":"Demo","unexpected":true}`,
		"trailing value": `{"name":"Demo"} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			var target struct {
				Name string `json:"name"`
			}
			if err := DecodeStrict(strings.NewReader(input), &target); err == nil {
				t.Fatal("DecodeStrict unexpectedly succeeded")
			}
		})
	}
}

func TestDecodeSinglePreservesNumbersAndRejectsTrailingValues(t *testing.T) {
	value, err := DecodeSingle([]byte(`{"confidence":0.75}`))
	if err != nil {
		t.Fatalf("DecodeSingle: %v", err)
	}
	object := value.(map[string]any)
	if got := object["confidence"].(interface{ String() string }).String(); got != "0.75" {
		t.Fatalf("number = %q, want 0.75", got)
	}
	if _, err := DecodeSingle([]byte(`{} {}`)); !errors.Is(err, ErrTrailingJSON) {
		t.Fatalf("trailing value error = %v, want ErrTrailingJSON", err)
	}
}
