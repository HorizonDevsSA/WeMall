// Package clients holds gRPC client connections to downstream microservices.
package clients

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventoryv1 "github.com/wemall/gen/inventory/v1"
	notificationv1 "github.com/wemall/gen/notification/v1"
	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	paymentv1 "github.com/wemall/gen/payment/v1"
	reviewv1 "github.com/wemall/gen/review/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
)

// Clients bundles all downstream gRPC service clients.
type Clients struct {
	User         userv1.UserServiceClient
	Product      productv1.ProductServiceClient
	Order        orderv1.OrderServiceClient
	Seller       sellerv1.SellerServiceClient
	Inventory    inventoryv1.InventoryServiceClient
	Notification notificationv1.NotificationServiceClient
	Review       reviewv1.ReviewServiceClient
	Payment      paymentv1.PaymentServiceClient

	userConn         *grpc.ClientConn
	productConn      *grpc.ClientConn
	orderConn        *grpc.ClientConn
	sellerConn       *grpc.ClientConn
	inventoryConn    *grpc.ClientConn
	notificationConn *grpc.ClientConn
	reviewConn       *grpc.ClientConn
	paymentConn      *grpc.ClientConn
}

// New dials user, product, order, seller, inventory, notification, review, and payment services.
func New(userAddr, productAddr, orderAddr, sellerAddr, inventoryAddr, notificationAddr, reviewAddr, paymentAddr string) (*Clients, error) {
	dial := func(addr string) (*grpc.ClientConn, error) {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", addr, err)
		}
		return conn, nil
	}

	uConn, err := dial(userAddr)
	if err != nil {
		return nil, err
	}
	pConn, err := dial(productAddr)
	if err != nil {
		uConn.Close()
		return nil, err
	}
	oConn, err := dial(orderAddr)
	if err != nil {
		uConn.Close()
		pConn.Close()
		return nil, err
	}
	sConn, err := dial(sellerAddr)
	if err != nil {
		uConn.Close()
		pConn.Close()
		oConn.Close()
		return nil, err
	}
	iConn, err := dial(inventoryAddr)
	if err != nil {
		uConn.Close()
		pConn.Close()
		oConn.Close()
		sConn.Close()
		return nil, err
	}
	nConn, err := dial(notificationAddr)
	if err != nil {
		uConn.Close()
		pConn.Close()
		oConn.Close()
		sConn.Close()
		iConn.Close()
		return nil, err
	}
	rConn, err := dial(reviewAddr)
	if err != nil {
		uConn.Close()
		pConn.Close()
		oConn.Close()
		sConn.Close()
		iConn.Close()
		nConn.Close()
		return nil, err
	}
	payConn, err := dial(paymentAddr)
	if err != nil {
		uConn.Close()
		pConn.Close()
		oConn.Close()
		sConn.Close()
		iConn.Close()
		nConn.Close()
		rConn.Close()
		return nil, err
	}

	return &Clients{
		User:         userv1.NewUserServiceClient(uConn),
		Product:      productv1.NewProductServiceClient(pConn),
		Order:        orderv1.NewOrderServiceClient(oConn),
		Seller:       sellerv1.NewSellerServiceClient(sConn),
		Inventory:    inventoryv1.NewInventoryServiceClient(iConn),
		Notification: notificationv1.NewNotificationServiceClient(nConn),
		Review:       reviewv1.NewReviewServiceClient(rConn),
		Payment:      paymentv1.NewPaymentServiceClient(payConn),
		userConn:         uConn,
		productConn:      pConn,
		orderConn:        oConn,
		sellerConn:       sConn,
		inventoryConn:    iConn,
		notificationConn: nConn,
		reviewConn:       rConn,
		paymentConn:      payConn,
	}, nil
}

// Close shuts down all gRPC connections.
func (c *Clients) Close() {
		c.userConn.Close()
		c.productConn.Close()
		c.orderConn.Close()
		c.sellerConn.Close()
		c.inventoryConn.Close()
		c.notificationConn.Close()
		c.reviewConn.Close()
		c.paymentConn.Close()
}
