package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/wemall/notification-service/internal/config"
	"github.com/wemall/notification-service/internal/providers/email"
	"github.com/wemall/notification-service/internal/providers/email/templates"
)

func main() {
	to := flag.String("to", "akotoxmpimbo@gmail.com", "Recipient email address")
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Output loaded config info
	fmt.Printf("Testing SMTP Provider with config:\n")
	fmt.Printf("  Host: %s\n", cfg.SMTPHost)
	fmt.Printf("  Port: %s\n", cfg.SMTPPort)
	fmt.Printf("  User: %s\n", cfg.SMTPUser)
	fmt.Printf("  AppName: WeMall\n")
	fmt.Printf("  AppURL: https://wemall.co.zw\n")
	fmt.Printf("Sending all seller notification emails to: %s\n", *to)

	if cfg.SMTPUser == "" || cfg.SMTPPass == "" {
		fmt.Println("Error: SMTP_USER or SMTP_PASS is empty in configuration. Please make sure to export them or source them from root .env")
		os.Exit(1)
	}

	provider := email.NewSMTPProvider(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		"WeMall",
		"https://wemall.co.zw",
		logger,
	)

	// List of templates to send
	type emailJob struct {
		subject string
		body    string
	}

	jobs := []emailJob{}

	// 1. Welcome / Verification Email
	welcomeBody, err := templates.RenderTemplate(
		templates.WelcomeTemplate,
		"Test Seller",
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"VerifyURL": "https://wemall.co.zw/verify?token=test-verification-token",
		},
	)
	if err != nil {
		fmt.Printf("Error rendering WelcomeTemplate: %v\n", err)
		os.Exit(1)
	}
	jobs = append(jobs, emailJob{subject: "Welcome to WeMall! 🚀", body: welcomeBody})

	// 2. Password Reset
	resetBody, err := templates.RenderTemplate(
		templates.PasswordResetTemplate,
		"Test Seller",
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"ResetURL": "https://wemall.co.zw/reset-password?token=test-reset-token",
			"Expiry":   "2 hours",
		},
	)
	if err != nil {
		fmt.Printf("Error rendering PasswordResetTemplate: %v\n", err)
		os.Exit(1)
	}
	jobs = append(jobs, emailJob{subject: "Reset Your WeMall Password", body: resetBody})

	// 3. Password Changed
	changedBody, err := templates.RenderTemplate(
		templates.PasswordChangedTemplate,
		"Test Seller",
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"Device":    "MacBook Pro (macOS)",
			"IPAddress": "192.168.1.1",
			"Time":      "2026-06-28 15:22:54",
		},
	)
	if err != nil {
		fmt.Printf("Error rendering PasswordChangedTemplate: %v\n", err)
		os.Exit(1)
	}
	jobs = append(jobs, emailJob{subject: "Security Alert: Your WeMall Password Was Changed", body: changedBody})

	// 4. Low Stock Alert
	lowStockBody, err := templates.RenderTemplate(
		templates.LowStockTemplate,
		"Test Seller",
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"VariantSKU":     "TSHIRT-BLUE-L",
			"RemainingStock": int64(3),
		},
	)
	if err != nil {
		fmt.Printf("Error rendering LowStockTemplate: %v\n", err)
		os.Exit(1)
	}
	jobs = append(jobs, emailJob{subject: "Low Stock Warning ⚠️", body: lowStockBody})

	// 5. Store Status Update
	statusBody, err := templates.RenderTemplate(
		templates.StoreStatusChangedTemplate,
		"Test Seller",
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"StoreName": "Super Wear Zimbabwe",
			"Status":    "verified",
			"Reason":    "All business credentials approved by the admin team.",
		},
	)
	if err != nil {
		fmt.Printf("Error rendering StoreStatusChangedTemplate: %v\n", err)
		os.Exit(1)
	}
	jobs = append(jobs, emailJob{subject: "WeMall store status update", body: statusBody})

	// Send them all
	for i, job := range jobs {
		fmt.Printf("[%d/%d] Sending: %s... ", i+1, len(jobs), job.subject)
		err = provider.SendEmail(*to, job.subject, job.body)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("SUCCESS")
	}

	fmt.Println("All seller email notifications sent successfully!")
}
