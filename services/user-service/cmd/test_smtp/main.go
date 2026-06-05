package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wemall/user-service/internal/config"
	"github.com/wemall/user-service/internal/service"
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

	// Override config if not set in environment or just verify SMTP settings are present
	if cfg.SMTPUser == "" || cfg.SMTPPass == "" {
		fmt.Println("Warning: SMTP_USER or SMTP_PASS is empty in default config.")
		fmt.Println("Checking for local override...")
	}

	fmt.Printf("Using SMTP Config:\n")
	fmt.Printf("  Host: %s\n", cfg.SMTPHost)
	fmt.Printf("  Port: %s\n", cfg.SMTPPort)
	fmt.Printf("  User: %s\n", cfg.SMTPUser)
	fmt.Printf("  Pass: [REDACTED]\n")
	fmt.Printf("Sending test email to: %s\n", *to)

	emailSvc := service.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)

	fmt.Println("Sending welcome email...")
	err = emailSvc.SendSellerWelcomeEmail(*to, "Harare CBD Store Owner")
	if err != nil {
		fmt.Printf("Error sending welcome email: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Welcome email sent successfully!")

	fmt.Println("Sending store review notification (PROCESSING)...")
	err = emailSvc.SendSellerReviewNotification(*to, "Harare CBD Store Owner", "Harare CBD Store", "processing")
	if err != nil {
		fmt.Printf("Error sending processing notification: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Processing notification sent successfully!")

	fmt.Println("Sending store review notification (VERIFIED)...")
	err = emailSvc.SendSellerReviewNotification(*to, "Harare CBD Store Owner", "Harare CBD Store", "verified")
	if err != nil {
		fmt.Printf("Error sending verified notification: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Verified notification sent successfully!")

	fmt.Println("All test emails sent successfully!")
}
