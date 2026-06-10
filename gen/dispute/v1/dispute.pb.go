package disputev1

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DisputeStatus int32

const (
	DisputeStatus_DISPUTE_STATUS_UNSPECIFIED       DisputeStatus = 0
	DisputeStatus_DISPUTE_STATUS_OPEN              DisputeStatus = 1
	DisputeStatus_DISPUTE_STATUS_SELLER_REVIEW     DisputeStatus = 2
	DisputeStatus_DISPUTE_STATUS_ESCALATED         DisputeStatus = 3
	DisputeStatus_DISPUTE_STATUS_RESOLVED_REFUNDED DisputeStatus = 4
	DisputeStatus_DISPUTE_STATUS_RESOLVED_REJECTED DisputeStatus = 5
)

type Dispute struct {
	Id        string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OrderId   string                 `protobuf:"bytes,2,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	BuyerId   string                 `protobuf:"bytes,3,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
	SellerId  string                 `protobuf:"bytes,4,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	Reason    string                 `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`
	Status    DisputeStatus          `protobuf:"varint,6,opt,name=status,proto3,enum=dispute.v1.DisputeStatus" json:"status,omitempty"`
	CreatedAt *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
}

type DisputeMessage struct {
	Id           string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	DisputeId    string                 `protobuf:"bytes,2,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
	SenderId     string                 `protobuf:"bytes,3,opt,name=sender_id,json=senderId,proto3" json:"sender_id,omitempty"`
	Content      string                 `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
	EvidenceUrls []string               `protobuf:"bytes,5,rep,name=evidence_urls,json=evidenceUrls,proto3" json:"evidence_urls,omitempty"`
	CreatedAt    *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
}

type OpenDisputeRequest struct {
	OrderId      string   `protobuf:"bytes,1,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	Reason       string   `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
	EvidenceUrls []string `protobuf:"bytes,3,rep,name=evidence_urls,json=evidenceUrls,proto3" json:"evidence_urls,omitempty"`
	BuyerId      string   `protobuf:"bytes,4,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
}

type OpenDisputeResponse struct {
	Dispute *Dispute `protobuf:"bytes,1,opt,name=dispute,proto3" json:"dispute,omitempty"`
}

type ReplyToDisputeRequest struct {
	DisputeId    string   `protobuf:"bytes,1,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
	SenderId     string   `protobuf:"bytes,2,opt,name=sender_id,json=senderId,proto3" json:"sender_id,omitempty"`
	Message      string   `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	EvidenceUrls []string `protobuf:"bytes,4,rep,name=evidence_urls,json=evidenceUrls,proto3" json:"evidence_urls,omitempty"`
}

type ReplyToDisputeResponse struct {
	Message *DisputeMessage `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

type EscalateDisputeRequest struct {
	DisputeId string `protobuf:"bytes,1,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
}

type ResolveDisputeRequest struct {
	DisputeId  string        `protobuf:"bytes,1,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
	Resolution DisputeStatus `protobuf:"varint,2,opt,name=resolution,proto3,enum=dispute.v1.DisputeStatus" json:"resolution,omitempty"`
	AdminId    string        `protobuf:"bytes,3,opt,name=admin_id,json=adminId,proto3" json:"admin_id,omitempty"`
}

type GetDisputeRequest struct {
	DisputeId string `protobuf:"bytes,1,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
}

type ListDisputesRequest struct {
	UserId string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Role   string `protobuf:"bytes,2,opt,name=role,proto3" json:"role,omitempty"`
}

type ListDisputesResponse struct {
	Disputes []*Dispute `protobuf:"bytes,1,rep,name=disputes,proto3" json:"disputes,omitempty"`
}

type ListDisputeMessagesRequest struct {
	DisputeId string `protobuf:"bytes,1,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
}

type ListDisputeMessagesResponse struct {
	Messages []*DisputeMessage `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
}

type DisputeServiceServer interface {
	OpenDispute(context.Context, *OpenDisputeRequest) (*OpenDisputeResponse, error)
	ReplyToDispute(context.Context, *ReplyToDisputeRequest) (*ReplyToDisputeResponse, error)
	EscalateDispute(context.Context, *EscalateDisputeRequest) (*Dispute, error)
	ResolveDispute(context.Context, *ResolveDisputeRequest) (*Dispute, error)
	GetDispute(context.Context, *GetDisputeRequest) (*Dispute, error)
	ListDisputes(context.Context, *ListDisputesRequest) (*ListDisputesResponse, error)
	ListDisputeMessages(context.Context, *ListDisputeMessagesRequest) (*ListDisputeMessagesResponse, error)
	mustEmbedUnimplementedDisputeServiceServer()
}

type UnimplementedDisputeServiceServer struct{}

func (UnimplementedDisputeServiceServer) OpenDispute(context.Context, *OpenDisputeRequest) (*OpenDisputeResponse, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) ReplyToDispute(context.Context, *ReplyToDisputeRequest) (*ReplyToDisputeResponse, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) EscalateDispute(context.Context, *EscalateDisputeRequest) (*Dispute, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) ResolveDispute(context.Context, *ResolveDisputeRequest) (*Dispute, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) GetDispute(context.Context, *GetDisputeRequest) (*Dispute, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) ListDisputes(context.Context, *ListDisputesRequest) (*ListDisputesResponse, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) ListDisputeMessages(context.Context, *ListDisputeMessagesRequest) (*ListDisputeMessagesResponse, error) {
	return nil, nil
}
func (UnimplementedDisputeServiceServer) mustEmbedUnimplementedDisputeServiceServer() {}

func RegisterDisputeServiceServer(s grpc.ServiceRegistrar, srv DisputeServiceServer) {
	// Not implemented mock
}
