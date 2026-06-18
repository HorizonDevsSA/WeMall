package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
	secret := "change_this_in_production_minimum_32_chars_long"
	claims := &Claims{
		UserID: "1234",
		Role:   "seller",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	
	fmt.Printf("Token Length: %d\n", len(tokenStr))

	parsedClaims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(tokenStr, parsedClaims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		fmt.Printf("Parse Error: %v\n", err)
	} else if !parsedToken.Valid {
		fmt.Printf("Token not valid\n")
	} else {
		fmt.Printf("Parsed UserID: %s\n", parsedClaims.UserID)
	}
}
