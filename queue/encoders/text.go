package encoders

import (
	"github.com/pkg/errors"
)

type Text struct{}

func (t Text) Encode(i any) ([]byte, error) {
	switch i := i.(type) {
	case []byte:
		return i, nil
	case string:
		return []byte(i), nil
	default:
		return nil, errors.Errorf("unknown type %T to encode with %T", i, t)
	}
}

func (Text) ContentType() string {
	return "text/plain"
}
