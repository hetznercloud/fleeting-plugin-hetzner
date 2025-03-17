package hetzner

import (
	"bytes"
	"encoding/json"
	"reflect"
)

type LaxStringList []string

var _ json.Unmarshaler = &LaxStringList{}

func (o *LaxStringList) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewBuffer(data))

	var v any
	if err := d.Decode(&v); err != nil {
		return err
	}

	switch typed := v.(type) {
	case string:
		*o = []string{typed}
	case []any:
		for _, itemI := range typed {
			if item, ok := itemI.(string); ok {
				*o = append(*o, item)
			} else {
				return &json.UnmarshalTypeError{
					Value: string(data),
					Type:  reflect.TypeOf(*o),
				}
			}
		}
	default:
		return &json.UnmarshalTypeError{
			Value: string(data),
			Type:  reflect.TypeOf(*o),
		}
	}

	return nil
}
