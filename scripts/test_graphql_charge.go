package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
	addr := flag.String("addr", "http://api-gateway:8080/query", "API Gateway address")
	msisdn := flag.String("msisdn", "0781920203", "MSISDN to charge")
	flag.Parse()

	// 1. Generate valid Buyer JWT token using server secret
	jwtSecret := []byte("change_this_in_production_minimum_32_chars_long")
	claims := &Claims{
		UserID: "test-buyer-123",
		Role:   "BUYER",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		panic(err)
	}
	fmt.Println("✔ Generated valid BUYER JWT Token")

	// 2. Prepare GraphQL Mutation
	query := fmt.Sprintf(`
		mutation {
			ecocashCharge(orderId: "test-order-777", msisdn: "%s", amountCents: 100, currency: "USD") {
				transaction {
					id
					status
					amountCents
					currency
					msisdn
				}
				statusMsg
			}
		}
	`, *msisdn)

	payload := map[string]string{
		"query": query,
	}
	bodyBytes, _ := json.Marshal(payload)

	// 3. Send HTTP Request to internal API Gateway
	fmt.Printf("▶ Sending ecocashCharge request for MSISDN %s to %s...\n", *msisdn, *addr)
	req, err := http.NewRequest("POST", *addr, bytes.NewReader(bodyBytes))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n--- Response ---\n")
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBytes, "", "  "); err == nil {
		fmt.Printf("Body:\n%s\n", prettyJSON.String())
	} else {
		fmt.Printf("Body: %s\n", string(respBytes))
	}
}
