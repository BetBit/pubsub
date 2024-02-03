package pubsub

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion7

type PubSubClient interface {
	Channel(ctx context.Context, opts ...grpc.CallOption) (PubSub_ChannelClient, error)
}

type pubSubClient struct {
	cc grpc.ClientConnInterface
}

func NewPubSubClient(cc grpc.ClientConnInterface) PubSubClient {
	return &pubSubClient{cc}
}

func (c *pubSubClient) Channel(ctx context.Context, opts ...grpc.CallOption) (PubSub_ChannelClient, error) {
	stream, err := c.cc.NewStream(ctx, &PubSub_ServiceDesc.Streams[0], "/proto.PubSub/Channel", opts...)
	if err != nil {
		return nil, err
	}
	x := &pubSubChannelClient{stream}
	return x, nil
}

type PubSub_ChannelClient interface {
	Send(*Event) error
	Recv() (*Event, error)
	grpc.ClientStream
}

type pubSubChannelClient struct {
	grpc.ClientStream
}

func (x *pubSubChannelClient) Send(m *Event) error {
	return x.ClientStream.SendMsg(m)
}

func (x *pubSubChannelClient) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type PubSubServer interface {
	Channel(PubSub_ChannelServer) error
	mustEmbedUnimplementedPubSubServer()
}

type UnimplementedPubSubServer struct {
}

func (UnimplementedPubSubServer) Channel(PubSub_ChannelServer) error {
	return status.Errorf(codes.Unimplemented, "method Channel not implemented")
}
func (UnimplementedPubSubServer) mustEmbedUnimplementedPubSubServer() {}

type UnsafePubSubServer interface {
	mustEmbedUnimplementedPubSubServer()
}

func RegisterPubSubServer(s grpc.ServiceRegistrar, srv PubSubServer) {
	s.RegisterService(&PubSub_ServiceDesc, srv)
}

func _PubSub_Channel_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(PubSubServer).Channel(&pubSubChannelServer{stream})
}

type PubSub_ChannelServer interface {
	Send(*Event) error
	Recv() (*Event, error)
	grpc.ServerStream
}

type pubSubChannelServer struct {
	grpc.ServerStream
}

func (x *pubSubChannelServer) Send(m *Event) error {
	return x.ServerStream.SendMsg(m)
}

func (x *pubSubChannelServer) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var PubSub_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.PubSub",
	HandlerType: (*PubSubServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Channel",
			Handler:       _PubSub_Channel_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "internal/proto/pubsub.proto",
}
