package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload shape
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
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

	fmt.Printf("Generated JWT: %s\n\n", tokenString)

	query := `
		mutation {
			ecocashCharge(orderId: "test-order-123", msisdn: "0781920203", amountCents: 100, currency: "USD") {
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
	`

	payload := map[string]string{
		"query": query,
	}
	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.15.240.45.232.nip.io/query", bytes.NewReader(bodyBytes))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(respBytes))
}
