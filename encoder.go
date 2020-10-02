package autonats

import (
	"encoding/json"
	"errors"
	"github.com/nats-io/nats.go"
	"strings"
)

// Unique encoder id
const ENCODER = "AUTONATS_ENCODER"

func init() {
	// Register our encoder
	nats.RegisterEncoder(ENCODER, &Encoder{})
}

type Reply struct {
	Data  []byte `json:"d,omitempty"`
	Error []byte `json:"error,omitempty"`
}

type Encoder struct{}

// Implement (nats.Encoder).Encode
func (enc *Encoder) Encode(subject string, v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Implement (nats.Encoder).Decode
func (enc *Encoder) Decode(subject string, data []byte, vPtr interface{}) error {
	var reply Reply

	if err := json.Unmarshal(data, &reply); err != nil {
		return err
	}

	// our way of checking if err != nil
	if len(reply.Error) > 0 {
		return errors.New(string(reply.Error))
	}

	if vPtr == nil {
		return nil
	}

	// this block is mostly borrowed from the built-in JSON Encoder
	switch arg := vPtr.(type) {
	case *string:
		str := string(reply.Data)
		if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
			*arg = str[1 : len(str)-1]
		} else {
			*arg = str
		}

	case *[]byte:
		*arg = reply.Data

	default:
		return json.Unmarshal(reply.Data, vPtr)
	}

	return nil
}
