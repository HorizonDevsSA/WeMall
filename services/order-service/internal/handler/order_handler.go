package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/wemall/gen/order/v1"
	"github.com/wemall/order-service/internal/service"
)

type OrderHandler struct {
	orderv1.UnimplementedOrderServiceServer
	cartSvc  *service.CartService
	orderSvc *service.OrderService
}

func NewOrderHandler(cartSvc *service.CartService, orderSvc *service.OrderService) *OrderHandler {
	return &OrderHandler{
		cartSvc:  cartSvc,
		orderSvc: orderSvc,
	}
}

// ── Cart RPCs ────────────────────────────────────────────────────────────────

func (h *OrderHandler) GetCart(ctx context.Context, req *orderv1.GetCartRequest) (*orderv1.Cart, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	cart, err := h.cartSvc.GetCart(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get cart")
	}
	return cart, nil
}

func (h *OrderHandler) AddToCart(ctx context.Context, req *orderv1.AddToCartRequest) (*orderv1.Cart, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	vid, err := uuid.Parse(req.VariantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid variant id")
	}
	cart, err := h.cartSvc.AddToCart(ctx, uid, vid, req.Quantity)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add to cart")
	}
	return cart, nil
}

func (h *OrderHandler) UpdateCartItem(ctx context.Context, req *orderv1.UpdateCartItemRequest) (*orderv1.Cart, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	iid, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}
	cart, err := h.cartSvc.UpdateCartItem(ctx, uid, iid, req.Quantity)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update cart item")
	}
	return cart, nil
}

func (h *OrderHandler) RemoveCartItem(ctx context.Context, req *orderv1.RemoveCartItemRequest) (*orderv1.Cart, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	iid, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item id")
	}
	cart, err := h.cartSvc.RemoveCartItem(ctx, uid, iid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove cart item")
	}
	return cart, nil
}

func (h *OrderHandler) ClearCart(ctx context.Context, req *orderv1.ClearCartRequest) (*orderv1.Cart, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	cart, err := h.cartSvc.ClearCart(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to clear cart")
	}
	return cart, nil
}

// ── Order RPCs ───────────────────────────────────────────────────────────────

func (h *OrderHandler) Checkout(ctx context.Context, req *orderv1.CheckoutRequest) (*orderv1.Order, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	order, err := h.orderSvc.Checkout(ctx, uid, req.Input)
	if err != nil {
		return nil, status.Error(codes.Internal, "checkout failed")
	}
	return order, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.Order, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	oid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}
	order, err := h.orderSvc.GetOrder(ctx, oid, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	return order, nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	orders, total, nextToken, err := h.orderSvc.ListOrders(ctx, uid, req.PageSize, req.PageToken)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list orders")
	}
	return &orderv1.ListOrdersResponse{
		Orders:        orders,
		NextPageToken: nextToken,
		Total:         total,
	}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.Order, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	oid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}
	order, err := h.orderSvc.CancelOrder(ctx, oid, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cancel order")
	}
	return order, nil
}
