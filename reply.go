package autonats

import (
	"errors"
	"github.com/json-iterator/go"
	"sync"
)

var replyPool = &sync.Pool{
	New: func() interface{} {
		return new(Reply)
	},
}

func GetReply() *Reply {
	return replyPool.Get().(*Reply)
}

func PutReply(reply *Reply) {
	reply.Reset()
	replyPool.Put(reply)
}

type Reply struct {
	Data  []byte `json:"d,omitempty"`
	Error []byte `json:"e,omitempty"`
}

func (r *Reply) MarshalBinary() ([]byte, error) {
	return jsoniter.Marshal(r)
}

func (r *Reply) UnmarshalBinary(data []byte) error {
	return jsoniter.Unmarshal(data, r)
}

func (r *Reply) WriteString(data string) {
	r.Data = []byte(data)
}

func (r *Reply) SetData(data []byte) {
	r.Data = data
}

func (r *Reply) MarshalAndSetData(data interface{}) error {
	var err error
	r.Data, err = jsoniter.Marshal(data)
	return err
}

func (r *Reply) GetError() error {
	if r.Error == nil {
		return nil
	} else {
		return errors.New(string(r.Error))
	}
}

func (r *Reply) GetDataAsString() string {
	return string(r.Data)
}

func (r *Reply) UnmarshalData(vPtr interface{}) error {
	if vPtr == nil {
		return nil
	}

	return jsoniter.Unmarshal(r.Data, vPtr)
}

func (r *Reply) Reset() {
	r.Data = nil
	r.Error = nil
}
