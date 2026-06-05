package templates

import (
	"bytes"
	"html/template"
	"time"
)

type EmailData struct {
	AppName       string
	AppURL        string
	LogoURL       string
	Year          int
	RecipientName string
	Extra         map[string]interface{}
}

func RenderTemplate(tmplStr string, recipientName, appName, appURL string, extra map[string]interface{}) (string, error) {
	data := EmailData{
		AppName:       appName,
		AppURL:        appURL,
		LogoURL:       appURL + "/logo.png",
		Year:          time.Now().Year(),
		RecipientName: recipientName,
		Extra:         extra,
	}
	if data.Extra == nil {
		data.Extra = make(map[string]interface{})
	}

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

// Base Styles (Glassmorphism inspired dark theme)
const BaseStyles = `
  body { margin: 0; padding: 0; background-color: #0f0f13; font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
  .wrapper { max-width: 600px; margin: 0 auto; padding: 40px 20px; }
  .card { background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%); border-radius: 16px; overflow: hidden; box-shadow: 0 25px 50px rgba(0,0,0,0.5); border: 1px solid rgba(255,255,255,0.08); }
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
`

// Welcome Template
const WelcomeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Welcome to {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
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
          Thank you for signing up at <span class="highlight">{{.AppName}}</span>.
          We are thrilled to have you join our community.
        </p>
        <p class="text">
          Please verify your account by clicking the button below:
        </p>
        <div class="cta-container">
          <a href="{{index .Extra "VerifyURL"}}" class="cta-btn">Confirm Account →</a>
        </div>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc. Harare, Zimbabwe.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Password Reset Template
const PasswordResetTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Reset Password — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Security Verification</div>
      </div>
      <div class="body">
        <h1 class="greeting">Reset Your Password 🔐</h1>
        <p class="text">
          Hello {{.RecipientName}}, we received a request to reset your password.
          Click the button below to specify a new password. This link will expire in <span class="highlight">{{index .Extra "Expiry"}}</span>.
        </p>
        <div class="cta-container">
          <a href="{{index .Extra "ResetURL"}}" class="cta-btn">Reset Password →</a>
        </div>
        <p class="text" style="font-size: 13px; color: #64748b;">
          If you did not request this, you can ignore this email safely.
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc. All rights reserved.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Password Changed Alert Template
const PasswordChangedTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Password Changed — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Security Alert</div>
      </div>
      <div class="body">
        <h1 class="greeting">Security Alert: Password Changed ⚠️</h1>
        <p class="text">
          Hi {{.RecipientName}}, the password for your account has been successfully changed.
        </p>
        <p class="text">
          <strong>Device:</strong> {{index .Extra "Device"}}<br>
          <strong>IP Address:</strong> {{index .Extra "IPAddress"}}<br>
          <strong>Time:</strong> {{index .Extra "Time"}}
        </p>
        <p class="text" style="color: #f43f5e;">
          If you did not authorize this change, please contact security support immediately.
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Payment Completed Template
const PaymentCompletedTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Payment Confirmation — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Payment Confirmation</div>
      </div>
      <div class="body">
        <h1 class="greeting">Order Confirmed! 🛒</h1>
        <p class="text">
          Hi {{.RecipientName}}, your payment has been processed successfully. Your order is now being prepared.
        </p>
        <p class="text">
          <strong>Order Number:</strong> {{index .Extra "OrderNumber"}}<br>
          <strong>Total Paid:</strong> {{index .Extra "Total"}} {{index .Extra "Currency"}}
        </p>
        <hr class="divider">
        <p class="text">
          We have sent a copy of your receipt to your account.
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Order Shipped Template
const OrderShippedTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Order Shipped — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Delivery Update</div>
      </div>
      <div class="body">
        <h1 class="greeting">Your Order is on the Way! 🚚</h1>
        <p class="text">
          Great news, {{.RecipientName}}! Your order has been shipped.
        </p>
        <p class="text">
          <strong>Order Number:</strong> {{index .Extra "OrderNumber"}}<br>
          <strong>Carrier:</strong> {{index .Extra "Carrier"}}<br>
          <strong>Tracking Number:</strong> <span class="highlight">{{index .Extra "TrackingNumber"}}</span>
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Low Stock Template
const LowStockTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Low Stock Alert — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Inventory Warning</div>
      </div>
      <div class="body">
        <h1 class="greeting">Low Stock Alert ⚠️</h1>
        <p class="text">
          Your inventory levels for variant <strong>{{index .Extra "VariantSKU"}}</strong> has fallen below the threshold.
        </p>
        <p class="text">
          <strong>Remaining Stock:</strong> <span style="color: #ef4444; font-weight: 700;">{{index .Extra "RemainingStock"}}</span>
        </p>
        <p class="text">
          Please restock this item soon to avoid missing out on potential sales.
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Seller Services.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Store Update Template
const StoreUpdateTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Store Update — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Followed Store Update</div>
      </div>
      <div class="body">
        <h1 class="greeting">New post from {{index .Extra "StoreName"}} 🏪</h1>
        <p class="text">
          A store you follow has added a new product or promotion:
        </p>
        <p class="text">
          <strong>Product:</strong> {{index .Extra "ProductTitle"}}<br>
          <strong>Price:</strong> {{index .Extra "Price"}} USD
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Refund Issued Template
const RefundIssuedTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Refund Issued — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Refund Processed</div>
      </div>
      <div class="body">
        <h1 class="greeting">Refund Confirmation 💸</h1>
        <p class="text">
          Hello {{.RecipientName}}, a refund has been issued for your order.
        </p>
        <p class="text">
          <strong>Order Number:</strong> {{index .Extra "OrderNumber"}}<br>
          <strong>Refund Amount:</strong> {{index .Extra "RefundAmount"}} USD<br>
          <strong>Expected Arrival (ETA):</strong> {{index .Extra "ETA"}}
        </p>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Store Status Changed Template
const StoreStatusChangedTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Store Status Update — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Store Status Update</div>
      </div>
      <div class="body">
        <h1 class="greeting">Store Status Update</h1>
        <p class="text">
          Hi {{.RecipientName}}, the status of your store <strong>{{index .Extra "StoreName"}}</strong> has been updated.
        </p>
        <div style="text-align: center; margin: 24px 0;">
          <span class="status-badge status-{{index .Extra "Status"}}">{{index .Extra "Status"}}</span>
        </div>
        {{if index .Extra "Reason"}}
        <p class="text">
          <strong>Reason:</strong> {{index .Extra "Reason"}}
        </p>
        {{end}}
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

// Back in stock template
const RestockedTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Back in Stock — {{.AppName}}</title>
  <style>` + BaseStyles + `</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="header">
        <div class="logo">We<span>Mall</span></div>
        <div class="tagline">Stock Alert</div>
      </div>
      <div class="body">
        <h1 class="greeting">It's Back! 🎉</h1>
        <p class="text">
          Hi {{.RecipientName}}, an item you were waiting for is back in stock:
        </p>
        <p class="text">
          <strong>{{index .Extra "ProductTitle"}}</strong> is now available.
        </p>
        <div class="cta-container">
          <a href="{{index .Extra "URL"}}" class="cta-btn">Buy Now →</a>
        </div>
      </div>
      <div class="footer">
        <p class="footer-text">© {{.Year}} {{.AppName}} Inc.</p>
      </div>
    </div>
  </div>
</body>
</html>`

func GetTemplateByEvent(event string) string {
	switch event {
	case "wemall.user.registered":
		return WelcomeTemplate
	case "wemall.user.password_reset":
		return PasswordResetTemplate
	case "wemall.user.password_changed":
		return PasswordChangedTemplate
	case "wemall.payment.completed":
		return PaymentCompletedTemplate
	case "wemall.order.shipped":
		return OrderShippedTemplate
	case "wemall.inventory.low_stock":
		return LowStockTemplate
	case "wemall.store.post_update":
		return StoreUpdateTemplate
	case "wemall.payment.refunded":
		return RefundIssuedTemplate
	case "wemall.seller.status_changed":
		return StoreStatusChangedTemplate
	case "wemall.inventory.restocked":
		return RestockedTemplate
	default:
		return ""
	}
}
