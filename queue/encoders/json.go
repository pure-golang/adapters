package encoders

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type JSON struct{}

func (JSON) Encode(i any) ([]byte, error) {
	b, err := json.Marshal(i)
	return b, errors.Wrapf(err, "marshal %T: %+v", i, i)
}

func (JSON) ContentType() string {
	return "application/json"
}
