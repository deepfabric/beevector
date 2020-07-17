package codec

import (
	"fmt"

	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/util/protoc"
)

type clientCodec struct {
}

func (c *clientCodec) Decode(in *goetty.ByteBuf) (bool, interface{}, error) {
	data := in.GetMarkedRemindData()
	resp := &rpcpb.Response{}
	err := resp.Unmarshal(data)
	if err != nil {
		return false, nil, err
	}

	in.MarkedBytesReaded()
	return true, resp, nil
}

func (c *clientCodec) Encode(data interface{}, out *goetty.ByteBuf) error {
	if req, ok := data.(*rpcpb.Request); ok {
		index := out.GetWriteIndex()
		size := req.Size()
		out.Expansion(size)
		protoc.MustMarshalTo(req, out.RawBuf()[index:index+size])
		out.SetWriterIndex(index + size)
		return nil
	}

	return fmt.Errorf("not support %T %+v", data, data)
}
