// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BlobServiceClient is the client API for BlobService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlobServiceClient interface {
	GetBlobSidecars(ctx context.Context, in *GetBlobSidecarsRequest, opts ...grpc.CallOption) (*GetBlobSidecarsResponse, error)
}

type blobServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBlobServiceClient(cc grpc.ClientConnInterface) BlobServiceClient {
	return &blobServiceClient{cc}
}

func (c *blobServiceClient) GetBlobSidecars(ctx context.Context, in *GetBlobSidecarsRequest, opts ...grpc.CallOption) (*GetBlobSidecarsResponse, error) {
	out := new(GetBlobSidecarsResponse)
	err := c.cc.Invoke(ctx, "/user.BlobService/GetBlobSidecars", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlobServiceServer is the server API for BlobService service.
// All implementations must embed UnimplementedBlobServiceServer
// for forward compatibility
type BlobServiceServer interface {
	GetBlobSidecars(context.Context, *GetBlobSidecarsRequest) (*GetBlobSidecarsResponse, error)
	mustEmbedUnimplementedBlobServiceServer()
}

// UnimplementedBlobServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBlobServiceServer struct {
}

func (UnimplementedBlobServiceServer) GetBlobSidecars(context.Context, *GetBlobSidecarsRequest) (*GetBlobSidecarsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlobSidecars not implemented")
}
func (UnimplementedBlobServiceServer) mustEmbedUnimplementedBlobServiceServer() {}

// UnsafeBlobServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlobServiceServer will
// result in compilation errors.
type UnsafeBlobServiceServer interface {
	mustEmbedUnimplementedBlobServiceServer()
}

func RegisterBlobServiceServer(s grpc.ServiceRegistrar, srv BlobServiceServer) {
	s.RegisterService(&BlobService_ServiceDesc, srv)
}

func _BlobService_GetBlobSidecars_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBlobSidecarsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobServiceServer).GetBlobSidecars(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.BlobService/GetBlobSidecars",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobServiceServer).GetBlobSidecars(ctx, req.(*GetBlobSidecarsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BlobService_ServiceDesc is the grpc.ServiceDesc for BlobService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlobService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "user.BlobService",
	HandlerType: (*BlobServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBlobSidecars",
			Handler:    _BlobService_GetBlobSidecars_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/blob.proto",
}
