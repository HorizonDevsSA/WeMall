package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"
)

type ChargeRequest struct {
	ClientCorrelator string `json:"clientCorrelator"`
	NotifyURL        string `json:"notifyUrl"`
	ReferenceCode    string `json:"referenceCode"`
	TranType         string `json:"tranType"`
	EndUserID        string `json:"endUserId"`
	Remark           string `json:"remark"`
	Merchant         struct {
		MerchantCode   string `json:"merchantCode"`
		MerchantPin    string `json:"merchantPin"`
		MerchantNumber string `json:"merchantNumber"`
		TerminalID     string `json:"terminalID"`
		Location       string `json:"location"`
		Currency       string `json:"currency"`
		CountryCode    string `json:"countryCode"`
		MerchantName   string `json:"merchantName"`
		SuperMerchant  string `json:"superMerchantName,omitempty"`
	} `json:"chargeMetaData"`
	Amount struct {
		ChargingInformation struct {
			Amount      string `json:"amount"`
			Currency    string `json:"currency"`
			Description string `json:"description"`
		} `json:"charginginformation"`
	} `json:"paymentAmount"`
}

type ChargeResponse struct {
	ClientCorrelator string `json:"clientCorrelator"`
	TransactionID    string `json:"transactionId"`
	ReferenceCode    string `json:"referenceCode"`
	StatusCode       string `json:"statusCode"`
	StatusMessage    string `json:"statusMessage"`
	Description      string `json:"description"`
	EndUserID        string `json:"endUserId"`
	PaymentAmount    interface{} `json:"paymentAmount"`
}

type LookupResponse struct {
	ClientCorrelator string `json:"clientCorrelator"`
	TransactionID    string `json:"transactionId"`
	StatusCode       string `json:"statusCode"`
	StatusMessage    string `json:"statusMessage"`
	Description      string `json:"description"`
	TranType         string `json:"tranType"`
}

func main() {
	mux := http.NewServeMux()

	// 1. Charge handler (POST /transactions)
	mux.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		secretHeader := r.Header.Get("X-WeMall-Proxy-Secret")
		log.Printf("Received Charge request. X-WeMall-Proxy-Secret Header: '%s'", secretHeader)

		var req ChargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ChargeResponse{
				StatusCode:    "E001",
				StatusMessage: "Invalid Request Format",
			})
			return
		}

		log.Printf("Received charge request for endUserId: %s, clientCorrelator: %s, merchantPin: %s",
			req.EndUserID, req.ClientCorrelator, req.Merchant.MerchantPin)

		// Determine mock status based on the PIN Matrix
		statusCode := "0"
		statusMessage := "Transaction Successful"

		switch req.Merchant.MerchantPin {
		case "0000":
			statusCode = "0"
			statusMessage = "Transaction Successful"
		case "1111":
			statusCode = "E010"
			statusMessage = "Insufficient Customer Wallet Balance"
		case "2222":
			statusCode = "E003"
			statusMessage = "Invalid PIN"
		case "9999":
			statusCode = "E004"
			statusMessage = "Limit Exceeded"
		default:
			// Fallback to success if not specified
			statusCode = "0"
			statusMessage = "Transaction Successful"
		}

		resp := ChargeResponse{
			ClientCorrelator: req.ClientCorrelator,
			TransactionID:    fmt.Sprintf("mock-tx-%d", time.Now().UnixNano()),
			ReferenceCode:    fmt.Sprintf("MOCK-REF-%s", req.ClientCorrelator[:8]),
			StatusCode:       statusCode,
			StatusMessage:    statusMessage,
			Description:      "Mocked EcoCash Response for testing",
			EndUserID:        req.EndUserID,
			PaymentAmount:    req.Amount,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// 2. Refund handler (POST /transactions/refund)
	mux.HandleFunc("/transactions/refund", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		secretHeader := r.Header.Get("X-WeMall-Proxy-Secret")
		log.Printf("Received Refund request. X-WeMall-Proxy-Secret Header: '%s'", secretHeader)
		resp := ChargeResponse{
			StatusCode:    "0",
			StatusMessage: "SUCCESS",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// 3. Lookup handler (GET /transactions/{endUserId}/{clientCorrelator})
	// Matches /transactions/2637XXXXXXXX/orderID-ns
	lookupRe := regexp.MustCompile(`^/transactions/([^/]+)/([^/]+)$`)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		matches := lookupRe.FindStringSubmatch(r.URL.Path)
		if len(matches) != 3 {
			http.NotFound(w, r)
			return
		}
		endUserID := matches[1]
		clientCorrelator := matches[2]
		secretHeader := r.Header.Get("X-WeMall-Proxy-Secret")
		log.Printf("Received Lookup request for endUserID: %s, clientCorrelator: %s. X-WeMall-Proxy-Secret Header: '%s'",
			endUserID, clientCorrelator, secretHeader)

		resp := LookupResponse{
			ClientCorrelator: clientCorrelator,
			TransactionID:    fmt.Sprintf("mock-tx-lookup-%d", time.Now().UnixNano()),
			StatusCode:       "0",
			StatusMessage:    "Transaction Successful",
			Description:      "Mocked lookup response",
			TranType:         "MER",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	log.Printf("Starting EcoCash Mock Server on port 8019...")
	if err := http.ListenAndServe(":8019", mux); err != nil {
		log.Fatalf("Mock server failed: %v", err)
	}
}
