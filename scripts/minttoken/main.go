// minttoken mints a JWT for testing purposes.
// Usage: go run ./scripts/minttoken/ -id <uuid> -role <buyer|seller|admin>
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	id   := flag.String("id",   "", "user UUID")
	role := flag.String("role", "buyer", "role: buyer|seller|admin")
	flag.Parse()

	if *id == "" {
		fmt.Fprintln(os.Stderr, "usage: minttoken -id <uuid> -role <role>")
		os.Exit(1)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change_this_in_production_minimum_32_chars_long"
	}

	claims := jwt.MapClaims{
		"user_id": *id,
		"role":    *role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(tok)
}
