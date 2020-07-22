package codec

import (
	"fmt"

	"github.com/deepfabric/beevector/pkg/pb/rpcpb"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/util/protoc"
)

var (
	c = &codec{}
	// ServerDecoder server decoder
	ServerDecoder = goetty.NewIntLengthFieldBasedDecoderSize(c, 0, 0, 0, 100*1024*1024)
	// ServerEncoder server encoder
	ServerEncoder = goetty.NewIntLengthFieldBasedEncoder(c)

	cc = &clientCodec{}
	// ClientDecoder client decoder
	ClientDecoder = goetty.NewIntLengthFieldBasedDecoderSize(cc, 0, 0, 0, 100*1024*1024)
	// ClientEncoder client encoder
	ClientEncoder = goetty.NewIntLengthFieldBasedEncoder(cc)
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
