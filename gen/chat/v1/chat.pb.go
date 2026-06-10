package chatv1

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ThreadType int32

const (
	ThreadType_THREAD_TYPE_UNSPECIFIED   ThreadType = 0
	ThreadType_THREAD_TYPE_DIRECT        ThreadType = 1
	ThreadType_THREAD_TYPE_BROADCAST     ThreadType = 2
)

type MessageType int32

const (
	MessageType_MESSAGE_TYPE_UNSPECIFIED MessageType = 0
	MessageType_MESSAGE_TYPE_TEXT        MessageType = 1
	MessageType_MESSAGE_TYPE_IMAGE       MessageType = 2
	MessageType_MESSAGE_TYPE_VIDEO       MessageType = 3
	MessageType_MESSAGE_TYPE_DOCUMENT    MessageType = 4
	MessageType_MESSAGE_TYPE_AUDIO       MessageType = 5
	MessageType_MESSAGE_TYPE_PRODUCT     MessageType = 6
	MessageType_MESSAGE_TYPE_ORDER       MessageType = 7
	MessageType_MESSAGE_TYPE_PROMOTION   MessageType = 8
)

type Thread struct {
	Id        string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Type      ThreadType             `protobuf:"varint,2,opt,name=type,proto3,enum=chat.v1.ThreadType" json:"type,omitempty"`
	Title     string                 `protobuf:"bytes,3,opt,name=title,proto3" json:"title,omitempty"`
	BuyerId   string                 `protobuf:"bytes,4,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
	SellerId  string                 `protobuf:"bytes,5,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	OrderId   string                 `protobuf:"bytes,6,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	CreatedAt *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
}

type Message struct {
	Id          string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	ThreadId    string                 `protobuf:"bytes,2,opt,name=thread_id,json=threadId,proto3" json:"thread_id,omitempty"`
	SenderId    string                 `protobuf:"bytes,3,opt,name=sender_id,json=senderId,proto3" json:"sender_id,omitempty"`
	Type        MessageType            `protobuf:"varint,4,opt,name=type,proto3,enum=chat.v1.MessageType" json:"type,omitempty"`
	Content     string                 `protobuf:"bytes,5,opt,name=content,proto3" json:"content,omitempty"`
	MediaUrl    string                 `protobuf:"bytes,6,opt,name=media_url,json=mediaUrl,proto3" json:"media_url,omitempty"`
	ReferenceId string                 `protobuf:"bytes,7,opt,name=reference_id,json=referenceId,proto3" json:"reference_id,omitempty"`
	IsRead      bool                   `protobuf:"varint,8,opt,name=is_read,json=isRead,proto3" json:"is_read,omitempty"`
	CreatedAt   *timestamppb.Timestamp `protobuf:"bytes,9,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
}

type CreateThreadRequest struct {
	BuyerId  string `protobuf:"bytes,1,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
	SellerId string `protobuf:"bytes,2,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	OrderId  string `protobuf:"bytes,3,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
}

type CreateBroadcastGroupRequest struct {
	SellerId string `protobuf:"bytes,1,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	Title    string `protobuf:"bytes,2,opt,name=title,proto3" json:"title,omitempty"`
}

type SendMessageRequest struct {
	ThreadId    string      `protobuf:"bytes,1,opt,name=thread_id,json=threadId,proto3" json:"thread_id,omitempty"`
	SenderId    string      `protobuf:"bytes,2,opt,name=sender_id,json=senderId,proto3" json:"sender_id,omitempty"`
	Type        MessageType `protobuf:"varint,3,opt,name=type,proto3,enum=chat.v1.MessageType" json:"type,omitempty"`
	Content     string      `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
	MediaUrl    string      `protobuf:"bytes,5,opt,name=media_url,json=mediaUrl,proto3" json:"media_url,omitempty"`
	ReferenceId string      `protobuf:"bytes,6,opt,name=reference_id,json=referenceId,proto3" json:"reference_id,omitempty"`
}

type ListThreadsRequest struct {
	UserId string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Role   string `protobuf:"bytes,2,opt,name=role,proto3" json:"role,omitempty"`
}

type ListThreadsResponse struct {
	Threads []*Thread `protobuf:"bytes,1,rep,name=threads,proto3" json:"threads,omitempty"`
}

type ListMessagesRequest struct {
	ThreadId  string `protobuf:"bytes,1,opt,name=thread_id,json=threadId,proto3" json:"thread_id,omitempty"`
	PageToken string `protobuf:"bytes,2,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	PageSize  int32  `protobuf:"varint,3,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
}

type ListMessagesResponse struct {
	Messages      []*Message `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
	NextPageToken string     `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
}

type ChatServiceServer interface {
	CreateThread(context.Context, *CreateThreadRequest) (*Thread, error)
	CreateBroadcastGroup(context.Context, *CreateBroadcastGroupRequest) (*Thread, error)
	SendMessage(context.Context, *SendMessageRequest) (*Message, error)
	ListThreads(context.Context, *ListThreadsRequest) (*ListThreadsResponse, error)
	ListMessages(context.Context, *ListMessagesRequest) (*ListMessagesResponse, error)
	mustEmbedUnimplementedChatServiceServer()
}

type UnimplementedChatServiceServer struct{}

func (UnimplementedChatServiceServer) CreateThread(context.Context, *CreateThreadRequest) (*Thread, error) { return nil, nil }
func (UnimplementedChatServiceServer) CreateBroadcastGroup(context.Context, *CreateBroadcastGroupRequest) (*Thread, error) { return nil, nil }
func (UnimplementedChatServiceServer) SendMessage(context.Context, *SendMessageRequest) (*Message, error) { return nil, nil }
func (UnimplementedChatServiceServer) ListThreads(context.Context, *ListThreadsRequest) (*ListThreadsResponse, error) { return nil, nil }
func (UnimplementedChatServiceServer) ListMessages(context.Context, *ListMessagesRequest) (*ListMessagesResponse, error) { return nil, nil }
func (UnimplementedChatServiceServer) mustEmbedUnimplementedChatServiceServer() {}

func RegisterChatServiceServer(s grpc.ServiceRegistrar, srv ChatServiceServer) {
	// mock implementation for compilation
}
