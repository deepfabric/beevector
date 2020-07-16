package api

import (
	"fmt"

	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/util/protoc"
)

var (
	c       = &codec{}
	decoder = goetty.NewIntLengthFieldBasedDecoder(c)
	encoder = goetty.NewIntLengthFieldBasedEncoder(c)

	cc = &clientCodec{}
	// Decoder client decoder
	Decoder = goetty.NewIntLengthFieldBasedDecoder(cc)
	// Encoder client encoder
	Encoder = goetty.NewIntLengthFieldBasedEncoder(cc)
)

type codec struct {
}

func (c *codec) Decode(in *goetty.ByteBuf) (bool, interface{}, error) {
	data := in.GetMarkedRemindData()
	req := &rpcpb.Request{}
	err := req.Unmarshal(data)
	if err != nil {
		return false, nil, err
	}

	in.MarkedBytesReaded()
	return true, req, nil
}

func (c *codec) Encode(data interface{}, out *goetty.ByteBuf) error {
	if resp, ok := data.(*rpcpb.Response); ok {
		index := out.GetWriteIndex()
		size := resp.Size()
		out.Expansion(size)
		protoc.MustMarshalTo(resp, out.RawBuf()[index:index+size])
		out.SetWriterIndex(index + size)
		return nil
	}

	return fmt.Errorf("not support %T %+v", data, data)
}
