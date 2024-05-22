// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.26.1
// source: post.proto

package post

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	PostService_CreatePost_FullMethodName     = "/post.PostService/CreatePost"
	PostService_UpdatePost_FullMethodName     = "/post.PostService/UpdatePost"
	PostService_DeletePost_FullMethodName     = "/post.PostService/DeletePost"
	PostService_GetPostById_FullMethodName    = "/post.PostService/GetPostById"
	PostService_GetPostsOnPage_FullMethodName = "/post.PostService/GetPostsOnPage"
)

// PostServiceClient is the client API for PostService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PostServiceClient interface {
	CreatePost(ctx context.Context, in *TCreatePostRequest, opts ...grpc.CallOption) (*TCreatePostResponse, error)
	UpdatePost(ctx context.Context, in *TUpdatePostRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeletePost(ctx context.Context, in *TDeletePostRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetPostById(ctx context.Context, in *TGetPostByIdRequest, opts ...grpc.CallOption) (*TGetPostByIdResponse, error)
	GetPostsOnPage(ctx context.Context, in *TGetPostsOnPageRequest, opts ...grpc.CallOption) (*TGetPostsOnPageResponse, error)
}

type postServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPostServiceClient(cc grpc.ClientConnInterface) PostServiceClient {
	return &postServiceClient{cc}
}

func (c *postServiceClient) CreatePost(ctx context.Context, in *TCreatePostRequest, opts ...grpc.CallOption) (*TCreatePostResponse, error) {
	out := new(TCreatePostResponse)
	err := c.cc.Invoke(ctx, PostService_CreatePost_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *postServiceClient) UpdatePost(ctx context.Context, in *TUpdatePostRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, PostService_UpdatePost_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *postServiceClient) DeletePost(ctx context.Context, in *TDeletePostRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, PostService_DeletePost_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *postServiceClient) GetPostById(ctx context.Context, in *TGetPostByIdRequest, opts ...grpc.CallOption) (*TGetPostByIdResponse, error) {
	out := new(TGetPostByIdResponse)
	err := c.cc.Invoke(ctx, PostService_GetPostById_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *postServiceClient) GetPostsOnPage(ctx context.Context, in *TGetPostsOnPageRequest, opts ...grpc.CallOption) (*TGetPostsOnPageResponse, error) {
	out := new(TGetPostsOnPageResponse)
	err := c.cc.Invoke(ctx, PostService_GetPostsOnPage_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PostServiceServer is the server API for PostService service.
// All implementations must embed UnimplementedPostServiceServer
// for forward compatibility
type PostServiceServer interface {
	CreatePost(context.Context, *TCreatePostRequest) (*TCreatePostResponse, error)
	UpdatePost(context.Context, *TUpdatePostRequest) (*emptypb.Empty, error)
	DeletePost(context.Context, *TDeletePostRequest) (*emptypb.Empty, error)
	GetPostById(context.Context, *TGetPostByIdRequest) (*TGetPostByIdResponse, error)
	GetPostsOnPage(context.Context, *TGetPostsOnPageRequest) (*TGetPostsOnPageResponse, error)
	mustEmbedUnimplementedPostServiceServer()
}

// UnimplementedPostServiceServer must be embedded to have forward compatible implementations.
type UnimplementedPostServiceServer struct {
}

func (UnimplementedPostServiceServer) CreatePost(context.Context, *TCreatePostRequest) (*TCreatePostResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreatePost not implemented")
}
func (UnimplementedPostServiceServer) UpdatePost(context.Context, *TUpdatePostRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePost not implemented")
}
func (UnimplementedPostServiceServer) DeletePost(context.Context, *TDeletePostRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeletePost not implemented")
}
func (UnimplementedPostServiceServer) GetPostById(context.Context, *TGetPostByIdRequest) (*TGetPostByIdResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPostById not implemented")
}
func (UnimplementedPostServiceServer) GetPostsOnPage(context.Context, *TGetPostsOnPageRequest) (*TGetPostsOnPageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPostsOnPage not implemented")
}
func (UnimplementedPostServiceServer) mustEmbedUnimplementedPostServiceServer() {}

// UnsafePostServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PostServiceServer will
// result in compilation errors.
type UnsafePostServiceServer interface {
	mustEmbedUnimplementedPostServiceServer()
}

func RegisterPostServiceServer(s grpc.ServiceRegistrar, srv PostServiceServer) {
	s.RegisterService(&PostService_ServiceDesc, srv)
}

func _PostService_CreatePost_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TCreatePostRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostServiceServer).CreatePost(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostService_CreatePost_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostServiceServer).CreatePost(ctx, req.(*TCreatePostRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PostService_UpdatePost_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TUpdatePostRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostServiceServer).UpdatePost(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostService_UpdatePost_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostServiceServer).UpdatePost(ctx, req.(*TUpdatePostRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PostService_DeletePost_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TDeletePostRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostServiceServer).DeletePost(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostService_DeletePost_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostServiceServer).DeletePost(ctx, req.(*TDeletePostRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PostService_GetPostById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TGetPostByIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostServiceServer).GetPostById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostService_GetPostById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostServiceServer).GetPostById(ctx, req.(*TGetPostByIdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PostService_GetPostsOnPage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TGetPostsOnPageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostServiceServer).GetPostsOnPage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostService_GetPostsOnPage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostServiceServer).GetPostsOnPage(ctx, req.(*TGetPostsOnPageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PostService_ServiceDesc is the grpc.ServiceDesc for PostService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PostService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "post.PostService",
	HandlerType: (*PostServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreatePost",
			Handler:    _PostService_CreatePost_Handler,
		},
		{
			MethodName: "UpdatePost",
			Handler:    _PostService_UpdatePost_Handler,
		},
		{
			MethodName: "DeletePost",
			Handler:    _PostService_DeletePost_Handler,
		},
		{
			MethodName: "GetPostById",
			Handler:    _PostService_GetPostById_Handler,
		},
		{
			MethodName: "GetPostsOnPage",
			Handler:    _PostService_GetPostsOnPage_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "post.proto",
}

const (
	StatsService_GetPostStats_FullMethodName  = "/post.StatsService/GetPostStats"
	StatsService_GetTopPosts_FullMethodName   = "/post.StatsService/GetTopPosts"
	StatsService_GetTopAuthors_FullMethodName = "/post.StatsService/GetTopAuthors"
	StatsService_AddPost_FullMethodName       = "/post.StatsService/AddPost"
)

// StatsServiceClient is the client API for StatsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StatsServiceClient interface {
	GetPostStats(ctx context.Context, in *TGetPostStatsRequest, opts ...grpc.CallOption) (*TGetPostStatsResponse, error)
	GetTopPosts(ctx context.Context, in *TGetTopPostsRequest, opts ...grpc.CallOption) (*TGetTopPostsResponse, error)
	GetTopAuthors(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*TGetTopAuthorsResponse, error)
	AddPost(ctx context.Context, in *TAddPostRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type statsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStatsServiceClient(cc grpc.ClientConnInterface) StatsServiceClient {
	return &statsServiceClient{cc}
}

func (c *statsServiceClient) GetPostStats(ctx context.Context, in *TGetPostStatsRequest, opts ...grpc.CallOption) (*TGetPostStatsResponse, error) {
	out := new(TGetPostStatsResponse)
	err := c.cc.Invoke(ctx, StatsService_GetPostStats_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statsServiceClient) GetTopPosts(ctx context.Context, in *TGetTopPostsRequest, opts ...grpc.CallOption) (*TGetTopPostsResponse, error) {
	out := new(TGetTopPostsResponse)
	err := c.cc.Invoke(ctx, StatsService_GetTopPosts_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statsServiceClient) GetTopAuthors(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*TGetTopAuthorsResponse, error) {
	out := new(TGetTopAuthorsResponse)
	err := c.cc.Invoke(ctx, StatsService_GetTopAuthors_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statsServiceClient) AddPost(ctx context.Context, in *TAddPostRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, StatsService_AddPost_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StatsServiceServer is the server API for StatsService service.
// All implementations must embed UnimplementedStatsServiceServer
// for forward compatibility
type StatsServiceServer interface {
	GetPostStats(context.Context, *TGetPostStatsRequest) (*TGetPostStatsResponse, error)
	GetTopPosts(context.Context, *TGetTopPostsRequest) (*TGetTopPostsResponse, error)
	GetTopAuthors(context.Context, *emptypb.Empty) (*TGetTopAuthorsResponse, error)
	AddPost(context.Context, *TAddPostRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedStatsServiceServer()
}

// UnimplementedStatsServiceServer must be embedded to have forward compatible implementations.
type UnimplementedStatsServiceServer struct {
}

func (UnimplementedStatsServiceServer) GetPostStats(context.Context, *TGetPostStatsRequest) (*TGetPostStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPostStats not implemented")
}
func (UnimplementedStatsServiceServer) GetTopPosts(context.Context, *TGetTopPostsRequest) (*TGetTopPostsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTopPosts not implemented")
}
func (UnimplementedStatsServiceServer) GetTopAuthors(context.Context, *emptypb.Empty) (*TGetTopAuthorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTopAuthors not implemented")
}
func (UnimplementedStatsServiceServer) AddPost(context.Context, *TAddPostRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddPost not implemented")
}
func (UnimplementedStatsServiceServer) mustEmbedUnimplementedStatsServiceServer() {}

// UnsafeStatsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StatsServiceServer will
// result in compilation errors.
type UnsafeStatsServiceServer interface {
	mustEmbedUnimplementedStatsServiceServer()
}

func RegisterStatsServiceServer(s grpc.ServiceRegistrar, srv StatsServiceServer) {
	s.RegisterService(&StatsService_ServiceDesc, srv)
}

func _StatsService_GetPostStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TGetPostStatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatsServiceServer).GetPostStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StatsService_GetPostStats_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatsServiceServer).GetPostStats(ctx, req.(*TGetPostStatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StatsService_GetTopPosts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TGetTopPostsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatsServiceServer).GetTopPosts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StatsService_GetTopPosts_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatsServiceServer).GetTopPosts(ctx, req.(*TGetTopPostsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StatsService_GetTopAuthors_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatsServiceServer).GetTopAuthors(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StatsService_GetTopAuthors_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatsServiceServer).GetTopAuthors(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _StatsService_AddPost_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TAddPostRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatsServiceServer).AddPost(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StatsService_AddPost_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatsServiceServer).AddPost(ctx, req.(*TAddPostRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// StatsService_ServiceDesc is the grpc.ServiceDesc for StatsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StatsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "post.StatsService",
	HandlerType: (*StatsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPostStats",
			Handler:    _StatsService_GetPostStats_Handler,
		},
		{
			MethodName: "GetTopPosts",
			Handler:    _StatsService_GetTopPosts_Handler,
		},
		{
			MethodName: "GetTopAuthors",
			Handler:    _StatsService_GetTopAuthors_Handler,
		},
		{
			MethodName: "AddPost",
			Handler:    _StatsService_AddPost_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "post.proto",
}
