package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ecocashv1 "github.com/wemall/gen/ecocash/v1"
)

func main() {
	// Connect to ecocash-service gRPC server
	addr := "ecocash-service:9018"
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}
	defer conn.Close()

	client := ecocashv1.NewEcoCashServiceClient(conn)

	// Inputs for test
	orderID := "a9916a0a-7eeb-4cb5-9fad-da8f7695d7a0"
	msisdn := "0781920203"
	amountCents := int64(6099) // $60.99
	currency := "USD"

	fmt.Printf("Triggering ChargeCustomer on remote ecocash-service...\n")
	fmt.Printf("OrderID:      %s\n", orderID)
	fmt.Printf("MSISDN:       %s\n", msisdn)
	fmt.Printf("AmountCents:  %d ($%.2f)\n", amountCents, float64(amountCents)/100.0)
	fmt.Printf("Currency:     %s\n\n", currency)

	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()

	resp, err := client.ChargeCustomer(ctx, &ecocashv1.ChargeCustomerRequest{
		OrderId:     orderID,
		Msisdn:      msisdn,
		AmountCents: amountCents,
		Currency:    currency,
	})
	if err != nil {
		log.Fatalf("ChargeCustomer RPC error: %v", err)
	}

	fmt.Printf("=== Response Received ===\n")
	fmt.Printf("Status Message: %s\n", resp.StatusMsg)
	if resp.Transaction != nil {
		t := resp.Transaction
		fmt.Printf("Transaction ID:        %s\n", t.Id)
		fmt.Printf("Client Correlator:     %s\n", t.ClientCorrelator)
		fmt.Printf("Reference Code:        %s\n", t.ReferenceCode)
		fmt.Printf("Type:                  %v\n", t.TranType)
		fmt.Printf("Masked MSISDN:         %s\n", t.MsisdnMasked)
		fmt.Printf("Amount Cents:          %d\n", t.AmountCents)
		fmt.Printf("Currency:              %s\n", t.Currency)
		fmt.Printf("Status:                %v\n", t.Status)
		fmt.Printf("EcoCash Status Code:   %s\n", t.EcocashStatusCode)
		fmt.Printf("EcoCash Status Msg:    %s\n", t.EcocashStatusMsg)
		fmt.Printf("EcoCash Transaction ID: %s\n", t.EcocashTransactionId)
	} else {
		fmt.Printf("Transaction record is nil!\n")
	}
}
