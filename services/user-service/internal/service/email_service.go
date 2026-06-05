package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"
)

// EmailService handles all transactional email sending with enterprise-level HTML templates.
type EmailService struct {
	host     string
	port     string
	user     string
	password string
	from     string
	appName  string
	appURL   string
}

// NewEmailService creates a new email service with SMTP configuration.
func NewEmailService(host, port, user, password string) *EmailService {
	return &EmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     user,
		appName:  "WeMall",
		appURL:   "https://wemall.co.zw",
	}
}

// emailData holds all data for email templates
type emailData struct {
	AppName    string
	AppURL     string
	LogoURL    string
	Year       int
	RecipientName string
	// Additional fields per email type
	Extra map[string]interface{}
}

func (e *EmailService) baseData(name string) emailData {
	return emailData{
		AppName:       e.appName,
		AppURL:        e.appURL,
		LogoURL:       e.appURL + "/logo.png",
		Year:          time.Now().Year(),
		RecipientName: name,
		Extra:         map[string]interface{}{},
	}
}

// SendSellerWelcomeEmail sends a premium welcome email to a new seller after registration.
func (e *EmailService) SendSellerWelcomeEmail(toEmail, fullName string) error {
	data := e.baseData(fullName)
	data.Extra["loginURL"] = e.appURL + "/seller/login"

	subject := fmt.Sprintf("Welcome to %s — Your Store Journey Begins 🚀", e.appName)
	body, err := renderTemplate(sellerWelcomeTemplate, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	return e.send(toEmail, subject, body)
}

// SendSellerReviewNotification notifies a seller when their store review status changes.
func (e *EmailService) SendSellerReviewNotification(toEmail, fullName, storeName, newStatus string) error {
	data := e.baseData(fullName)
	data.Extra["storeName"] = storeName
	data.Extra["status"] = newStatus
	data.Extra["dashboardURL"] = e.appURL + "/seller/dashboard"

	var subject string
	switch strings.ToLower(newStatus) {
	case "processing":
		subject = fmt.Sprintf("Your store '%s' is under review 🔍", storeName)
	case "verified":
		subject = fmt.Sprintf("Congratulations! Your store '%s' is now verified ✅", storeName)
	case "suspended":
		subject = fmt.Sprintf("Important: Your store '%s' has been suspended ⚠️", storeName)
	default:
		subject = fmt.Sprintf("Update on your store '%s'", storeName)
	}

	body, err := renderTemplate(sellerStatusTemplate, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	return e.send(toEmail, subject, body)
}

// SendPasswordResetEmail sends a password reset link to a user.
func (e *EmailService) SendPasswordResetEmail(toEmail, fullName, resetToken string) error {
	data := e.baseData(fullName)
	data.Extra["resetURL"] = fmt.Sprintf("%s/seller/reset-password?token=%s", e.appURL, resetToken)
	data.Extra["expiryMinutes"] = 30

	subject := fmt.Sprintf("%s — Password Reset Request 🔐", e.appName)
	body, err := renderTemplate(passwordResetTemplate, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	return e.send(toEmail, subject, body)
}

// send sends an HTML email via SMTP using TLS.
func (e *EmailService) send(to, subject, htmlBody string) error {
	headers := map[string]string{
		"From":         fmt.Sprintf("%s <noreply@%s>", e.appName, "wemall.co.zw"),
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": `text/html; charset="UTF-8"`,
		"X-Mailer":     e.appName + " Mailer v1.0",
	}

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(k + ": " + v + "\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	auth := smtp.PlainAuth("", e.user, e.password, e.host)

	// Use TLS for Gmail
	tlsCfg := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         e.host,
	}

	addr := e.host + ":" + e.port
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		// Fallback to STARTTLS on port 587
		return e.sendSTARTTLS(to, subject, msg.Bytes(), auth)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(e.user); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg.Bytes()); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return w.Close()
}

func (e *EmailService) sendSTARTTLS(to, subject string, msg []byte, auth smtp.Auth) error {
	addr := e.host + ":" + e.port
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	tlsCfg := &tls.Config{ServerName: e.host}
	if err = client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(e.user); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return w.Close()
}

func renderTemplate(tmplStr string, data emailData) (string, error) {
	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ── HTML Email Templates ──────────────────────────────────────────────────────

const emailBaseStyles = `
  body { margin: 0; padding: 0; background-color: #0f0f13; font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
  .wrapper { max-width: 600px; margin: 0 auto; padding: 40px 20px; }
  .card { background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%); border-radius: 16px; overflow: hidden; box-shadow: 0 25px 50px rgba(0,0,0,0.5); }
  .header { background: linear-gradient(135deg, #6c63ff 0%, #a855f7 50%, #ec4899 100%); padding: 40px 48px 36px; text-align: center; }
  .logo { font-size: 32px; font-weight: 800; color: #ffffff; letter-spacing: -1px; }
  .logo span { color: rgba(255,255,255,0.7); }
  .tagline { color: rgba(255,255,255,0.85); font-size: 14px; margin-top: 8px; letter-spacing: 2px; text-transform: uppercase; }
  .body { padding: 48px; }
  .greeting { font-size: 26px; font-weight: 700; color: #f8fafc; margin: 0 0 8px; line-height: 1.3; }
  .text { color: #94a3b8; font-size: 16px; line-height: 1.7; margin: 16px 0; }
  .highlight { color: #a78bfa; font-weight: 600; }
  .cta-container { text-align: center; margin: 36px 0; }
  .cta-btn { display: inline-block; background: linear-gradient(135deg, #6c63ff, #a855f7); color: #ffffff; text-decoration: none; font-weight: 700; font-size: 16px; padding: 16px 40px; border-radius: 50px; letter-spacing: 0.5px; box-shadow: 0 8px 25px rgba(108,99,255,0.4); }
  .divider { border: none; border-top: 1px solid rgba(255,255,255,0.08); margin: 32px 0; }
  .feature-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin: 24px 0; }
  .feature-card { background: rgba(255,255,255,0.05); border-radius: 12px; padding: 20px; border: 1px solid rgba(255,255,255,0.08); }
  .feature-icon { font-size: 28px; margin-bottom: 10px; }
  .feature-title { color: #e2e8f0; font-weight: 600; font-size: 14px; margin-bottom: 6px; }
  .feature-desc { color: #64748b; font-size: 13px; line-height: 1.5; }
  .status-badge { display: inline-block; padding: 6px 18px; border-radius: 50px; font-weight: 700; font-size: 13px; text-transform: uppercase; letter-spacing: 1px; }
  .status-processing { background: rgba(251,191,36,0.15); color: #fbbf24; border: 1px solid rgba(251,191,36,0.3); }
  .status-verified { background: rgba(52,211,153,0.15); color: #34d399; border: 1px solid rgba(52,211,153,0.3); }
  .status-suspended { background: rgba(239,68,68,0.15); color: #ef4444; border: 1px solid rgba(239,68,68,0.3); }
  .footer { padding: 32px 48px; background: rgba(0,0,0,0.2); text-align: center; }
  .footer-text { color: #475569; font-size: 13px; line-height: 1.6; }
  .footer-link { color: #6c63ff; text-decoration: none; }
  .social-links { margin: 16px 0; }
  .social-link { display: inline-block; margin: 0 8px; color: #64748b; font-size: 13px; text-decoration: none; }
`

const sellerWelcomeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Welcome to {{.AppName}}</title>
  <style>` + emailBaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Zimbabwe's Premier Marketplace</div>
      </div>
      <div class="body">
        <h1 class="greeting">Welcome aboard, {{.RecipientName}}! 🎉</h1>
        <p class="text">
          You've just joined <span class="highlight">{{.AppName}}</span> — Zimbabwe's fastest-growing digital marketplace. 
          Your seller account has been created and your store is now in the review queue.
        </p>

        <div class="feature-grid">
          <div class="feature-card">
            <div class="feature-icon">🏪</div>
            <div class="feature-title">Open Your Store</div>
            <div class="feature-desc">Set up your storefront with a logo, banner, and description to attract buyers.</div>
          </div>
          <div class="feature-card">
            <div class="feature-icon">📦</div>
            <div class="feature-title">List Products</div>
            <div class="feature-desc">Add your products with variants, pricing, and high-quality images.</div>
          </div>
          <div class="feature-card">
            <div class="feature-icon">📍</div>
            <div class="feature-title">Local Discovery</div>
            <div class="feature-desc">Buyers near you can discover your store using our geolocation features.</div>
          </div>
          <div class="feature-card">
            <div class="feature-icon">💳</div>
            <div class="feature-title">Fast Payouts</div>
            <div class="feature-desc">Get paid directly to your account in USD or ZWG after each sale.</div>
          </div>
        </div>

        <hr class="divider">

        <p class="text">
          Your account is currently under review. Our team will verify your store details within 
          <span class="highlight">24–48 business hours</span>. You'll receive an email notification once your store is verified and ready to go live.
        </p>

        <div class="cta-container">
          <a href="{{index .Extra "loginURL"}}" class="cta-btn">Access Seller Dashboard →</a>
        </div>

        <p class="text" style="font-size: 14px; color: #475569;">
          Questions? Reply to this email or visit our 
          <a href="{{.AppURL}}/help" style="color: #6c63ff;">Help Center</a>. 
          We're here 24/7 to help you succeed.
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">
          You received this email because you registered as a seller on {{.AppName}}.<br>
          <a href="{{.AppURL}}" class="footer-link">{{.AppURL}}</a> · 
          <a href="{{.AppURL}}/privacy" class="footer-link">Privacy Policy</a> · 
          <a href="{{.AppURL}}/terms" class="footer-link">Terms of Service</a>
        </p>
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc. All rights reserved. Harare, Zimbabwe.</p>
      </div>
    </div>
  </div>
</body>
</html>`

const sellerStatusTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Store Status Update — {{.AppName}}</title>
  <style>` + emailBaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Store Status Update</div>
      </div>
      <div class="body">
        <h1 class="greeting">Store Update, {{.RecipientName}}</h1>
        
        <p class="text">
          There's an update on your store <strong style="color: #e2e8f0;">{{index .Extra "storeName"}}</strong>:
        </p>

        {{$status := index .Extra "status"}}
        {{if eq $status "processing"}}
        <div style="text-align: center; margin: 28px 0;">
          <span class="status-badge status-processing">🔍 Under Review</span>
        </div>
        <p class="text">
          Your store is currently <span class="highlight">being reviewed</span> by our team. This process typically 
          takes 24–48 business hours. We check your store details, product listings, and compliance with our 
          marketplace guidelines.
        </p>
        <p class="text">
          You'll be notified immediately once the review is complete. In the meantime, feel free to continue 
          setting up your store and adding products.
        </p>
        {{else if eq $status "verified"}}
        <div style="text-align: center; margin: 28px 0;">
          <span class="status-badge status-verified">✅ Verified</span>
        </div>
        <p class="text">
          🎊 Congratulations! Your store has been <span class="highlight">officially verified</span>. 
          You're now a trusted seller on {{.AppName}} and your products are live for buyers across Zimbabwe!
        </p>
        <p class="text">
          Your verified badge will appear on all your product listings, building buyer confidence and 
          increasing your chances of making sales.
        </p>
        {{else if eq $status "suspended"}}
        <div style="text-align: center; margin: 28px 0;">
          <span class="status-badge status-suspended">⚠️ Suspended</span>
        </div>
        <p class="text">
          We've had to temporarily <span style="color: #ef4444; font-weight: 600;">suspend</span> your store 
          due to a policy violation or compliance issue. Your listings are not visible to buyers during this period.
        </p>
        <p class="text">
          If you believe this is an error or want to appeal this decision, please contact our support team 
          immediately with your store details and any relevant documentation.
        </p>
        {{end}}

        <hr class="divider">

        <div class="cta-container">
          <a href="{{index .Extra "dashboardURL"}}" class="cta-btn">View Seller Dashboard →</a>
        </div>

        <p class="text" style="font-size: 14px; color: #475569;">
          Need help? Contact our seller support at 
          <a href="mailto:sellers@wemall.co.zw" style="color: #6c63ff;">sellers@wemall.co.zw</a>
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">
          This is an automated notification from {{.AppName}} Seller Platform.<br>
          <a href="{{.AppURL}}" class="footer-link">{{.AppURL}}</a> · 
          <a href="{{.AppURL}}/seller/support" class="footer-link">Seller Support</a>
        </p>
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc. All rights reserved. Harare, Zimbabwe.</p>
      </div>
    </div>
  </div>
</body>
</html>`

const passwordResetTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Password Reset — {{.AppName}}</title>
  <style>` + emailBaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Security Notice</div>
      </div>
      <div class="body">
        <h1 class="greeting">Password Reset Request 🔐</h1>
        <p class="text">
          Hi <strong style="color: #e2e8f0;">{{.RecipientName}}</strong>, we received a request to reset your 
          {{.AppName}} seller account password.
        </p>
        <p class="text">
          Click the button below to set a new password. This link expires in 
          <span class="highlight">{{index .Extra "expiryMinutes"}} minutes</span>.
        </p>

        <div class="cta-container">
          <a href="{{index .Extra "resetURL"}}" class="cta-btn">Reset Password →</a>
        </div>

        <hr class="divider">

        <p class="text" style="font-size: 14px; color: #475569;">
          If you didn't request a password reset, you can safely ignore this email. 
          Your password will not change unless you click the link above.
        </p>
        <p class="text" style="font-size: 13px; color: #334155;">
          For security, this link is single-use and will expire in {{index .Extra "expiryMinutes"}} minutes. 
          If you need a new link, go to the 
          <a href="{{.AppURL}}/seller/forgot-password" style="color: #6c63ff;">forgot password page</a>.
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">
          This security email was sent to you by {{.AppName}}.<br>
          <a href="{{.AppURL}}" class="footer-link">{{.AppURL}}</a>
        </p>
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc. All rights reserved. Harare, Zimbabwe.</p>
      </div>
    </div>
  </div>
</body>
</html>`
