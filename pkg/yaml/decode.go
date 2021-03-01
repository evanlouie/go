package yaml

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Decode a single or multi-document yaml body into a slice of interfaces.
// Will error if any document in the body is unable to be parsed.
func Decode(doc []byte) ([]interface{}, error) {
	var values []interface{}
	decoder := yaml.NewDecoder(bytes.NewReader(doc))
	// iterate until the decoder reaches an io.EOF
	for {
		var value interface{}
		err := decoder.Decode(&value)
		switch {
		case err == io.EOF:
			// base case: eof; return the value
			return values, nil
		case err != nil:
			// error case: return the error
			return nil, fmt.Errorf(`decoding yaml in yaml document %s`, string(doc))
		default:
			// recursive case: append the decoded value
			values = append(values, value)
		}
	}
}

// DecodeMaps will decode a single or multi-document yaml body into a slice of
// map[string]interface{}.
// This is primarily useful for when trying to decode large yaml strings
// containing one or more documents such as Kubernetes yaml.
func DecodeMaps(doc []byte) ([]map[string]interface{}, error) {
	// decode the docs
	values, err := Decode(doc)
	if err != nil {
		return nil, err
	}

	// reflect all top level objects into map[string]interface{}
	var maps []map[string]interface{}
	for _, value := range values {
		var decoded map[string]interface{}
		if value != nil {
			var ok bool
			decoded, ok = value.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf(`unable to reflect value %+v as a map[string]interface{}`, value)
			}
		}
		maps = append(maps, decoded)
	}

	return maps, nil
}
