// Package jsonutil provides shared JSON validation helpers for built-in tools.
package jsonutil

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RejectDuplicateObjectKeys returns an error when a JSON object contains duplicate keys.
func RejectDuplicateObjectKeys(text string) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return fmt.Errorf("invalid JSON: trailing data after %v", token)
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("invalid JSON object key: %v", keyToken)
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = true
			// 对 value 递归扫描，避免嵌套对象重复 key 在 map 解码前被静默覆盖。
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		return consumeJSONDelim(decoder, '}')
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		return consumeJSONDelim(decoder, ']')
	default:
		return fmt.Errorf("invalid JSON delimiter: %q", delim)
	}
}

func consumeJSONDelim(decoder *json.Decoder, want json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	got, ok := token.(json.Delim)
	if !ok || got != want {
		return fmt.Errorf("invalid JSON: expected %q, got %v", want, token)
	}
	return nil
}
