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
	fmt.Printf("Sending welcome email to: %s\n", *to)

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

	// Render welcome template
	body, err := templates.RenderTemplate(
		templates.WelcomeTemplate,
		"WeMall Customer",
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"VerifyURL": "https://wemall.co.zw/verify?token=test-verification-token",
		},
	)
	if err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		os.Exit(1)
	}

	err = provider.SendEmail(*to, "Welcome to WeMall! 🚀", body)
	if err != nil {
		fmt.Printf("Failed to send email: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Welcome email sent successfully!")
}
