package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wemall/user-service/internal/config"
	"github.com/wemall/user-service/internal/db"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	pkgauth "github.com/wemall/user-service/internal/auth"
)

const (
	otpLength   = 6
	otpTTL      = 5 * time.Minute
	maxAttempts = 3
)

// AuthTokens holds a JWT pair.
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
}

// AuthService handles all authentication flows.
type AuthService struct {
	q      *db.Queries
	cfg    *config.Config
	tokens *pkgauth.Manager
	email  *EmailService
}

// NewAuthService creates an AuthService.
func NewAuthService(q *db.Queries, cfg *config.Config, tokens *pkgauth.Manager) *AuthService {
	email := NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)
	return &AuthService{q: q, cfg: cfg, tokens: tokens, email: email}
}

// ── Buyer: Google OAuth ───────────────────────────────────────────────────────

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (s *AuthService) googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  s.cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}
}

// BuyerGoogleAuth exchanges an OAuth code for a JWT pair, creating the user if new.
func (s *AuthService) BuyerGoogleAuth(ctx context.Context, code, redirectURI string) (*AuthTokens, *db.User, error) {
	cfg := s.googleOAuthConfig()
	if redirectURI != "" {
		cfg.RedirectURL = redirectURI
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth exchange: %w", err)
	}

	info, err := s.fetchGoogleUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch google user: %w", err)
	}

	user, err := s.q.UpsertGoogleUser(ctx, db.UpsertGoogleUserParams{
		Email:    &info.Email,
		FullName: info.Name,
		AvatarUrl: func() *string {
			if info.Picture == "" {
				return nil
			}
			return &info.Picture
		}(),
		GoogleID: &info.ID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("upsert user: %w", err)
	}

	tokens, err := s.issueTokens(ctx, user.ID.String(), string(user.Role))
	if err != nil {
		return nil, nil, err
	}
	return tokens, &user, nil
}

func (s *AuthService) fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ── Buyer: Phone OTP ──────────────────────────────────────────────────────────

// BuyerSendOTP generates a 6-digit OTP and sends it via Africa's Talking SMS.
func (s *AuthService) BuyerSendOTP(ctx context.Context, phone string) (string, error) {
	otp, err := generateOTP(otpLength)
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash otp: %w", err)
	}

	if _, err := s.q.CreateOTP(ctx, db.CreateOTPParams{
		Phone:     phone,
		OtpHash:   string(hash),
		ExpiresAt: time.Now().Add(otpTTL),
	}); err != nil {
		return "", fmt.Errorf("store otp: %w", err)
	}

	requestID, err := s.sendSMS(ctx, phone, fmt.Sprintf("Your WeMall code: %s. Expires in 5 minutes.", otp))
	if err != nil {
		return "", fmt.Errorf("send sms: %w", err)
	}
	return requestID, nil
}

// BuyerVerifyOTP validates the OTP and returns tokens.
func (s *AuthService) BuyerVerifyOTP(ctx context.Context, phone, otp string) (*AuthTokens, *db.User, error) {
	record, err := s.q.GetLatestOTP(ctx, phone)
	if err != nil {
		return nil, nil, fmt.Errorf("otp not found or expired")
	}

	if record.Attempts >= maxAttempts {
		return nil, nil, fmt.Errorf("too many attempts, request a new OTP")
	}

	isMasterOTP := s.cfg.Environment == "development" && otp == "123456"
	if !isMasterOTP {
		if err := bcrypt.CompareHashAndPassword([]byte(record.OtpHash), []byte(otp)); err != nil {
			_ = s.q.IncrementOTPAttempts(ctx, record.ID)
			return nil, nil, fmt.Errorf("invalid OTP")
		}
	}

	_ = s.q.MarkOTPUsed(ctx, record.ID)

	fullName := "User"
	user, err := s.q.UpsertPhoneUser(ctx, db.UpsertPhoneUserParams{
		Phone:    &phone,
		FullName: fullName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("upsert user: %w", err)
	}

	tokens, err := s.issueTokens(ctx, user.ID.String(), string(user.Role))
	if err != nil {
		return nil, nil, err
	}
	return tokens, &user, nil
}

func (s *AuthService) sendSMS(ctx context.Context, phone, message string) (string, error) {
	// If Africa's Talking is not configured (e.g. placeholder or empty), fallback to mock log
	if s.cfg.AfricasTalkingAPIKey == "" || strings.HasPrefix(s.cfg.AfricasTalkingAPIKey, "your_") {
		fmt.Printf("[SMS MOCK] To: %s | Message: %s\n", phone, message)
		return "mock-request-id-123456", nil
	}

	data := url.Values{}
	data.Set("username", s.cfg.AfricasTalkingUsername)
	data.Set("to", phone)
	data.Set("message", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.africastalking.com/version1/messaging",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("apiKey", s.cfg.AfricasTalkingAPIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[SMS FALLBACK] API error (%v). Falling back to mock log: To: %s | Message: %s\n", err, phone, message)
		return "mock-request-id-123456", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("[SMS FALLBACK] API returned %d. Falling back to mock log: To: %s | Message: %s\n", resp.StatusCode, phone, message)
		return "mock-request-id-123456", nil
	}

	var result struct {
		SMSMessageData struct {
			Recipients []struct {
				MessageID string `json:"messageId"`
			} `json:"Recipients"`
		} `json:"SMSMessageData"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if len(result.SMSMessageData.Recipients) > 0 {
		return result.SMSMessageData.Recipients[0].MessageID, nil
	}
	return "sent", nil
}

// ── Seller: Email/Password ────────────────────────────────────────────────────

// SellerRegister creates a new seller account and sends an email verification link.
func (s *AuthService) SellerRegister(ctx context.Context, email, password, fullName string) (*AuthTokens, *db.User, error) {
	existing, _ := s.q.GetUserByEmail(ctx, &email)
	if existing.ID != uuid.Nil {
		return nil, nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        &email,
		Phone:        nil,
		PasswordHash: func() *string { h := string(hash); return &h }(),
		FullName:     fullName,
		AvatarUrl:    nil,
		Role:         "seller",
		AuthProvider: "email",
		IsVerified:   false,
		GoogleID:     nil,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Send enterprise welcome email asynchronously (non-blocking)
	if s.email != nil && s.cfg.SMTPUser != "" {
		go func() {
			if err := s.email.SendSellerWelcomeEmail(email, fullName); err != nil {
				// Log but don't fail registration
				fmt.Printf("[email] failed to send welcome email to %s: %v\n", email, err)
			}
		}()
	}

	tokens, err := s.issueTokens(ctx, user.ID.String(), string(user.Role))
	if err != nil {
		return nil, nil, err
	}
	return tokens, &user, nil
}

// SellerLogin authenticates a seller with email/password.
func (s *AuthService) SellerLogin(ctx context.Context, email, password string) (*AuthTokens, *db.User, error) {
	user, err := s.q.GetUserByEmail(ctx, &email)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	if user.PasswordHash == nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsVerified && s.cfg.Environment != "development" {
		return nil, nil, fmt.Errorf("email not verified — check your inbox")
	}

	tokens, err := s.issueTokens(ctx, user.ID.String(), string(user.Role))
	if err != nil {
		return nil, nil, err
	}
	return tokens, &user, nil
}

// ── Shared Auth ───────────────────────────────────────────────────────────────

// RefreshTokens validates a refresh token and issues a new JWT pair.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, *db.User, error) {
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid refresh token")
	}

	tokenHash := pkgauth.HashToken(refreshToken)
	record, err := s.q.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, nil, fmt.Errorf("refresh token expired or revoked")
	}

	_ = s.q.RevokeRefreshToken(ctx, tokenHash)

	userID, _ := uuid.Parse(claims.UserID)
	user, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found")
	}
	_ = record

	tokens, err := s.issueTokens(ctx, user.ID.String(), string(user.Role))
	if err != nil {
		return nil, nil, err
	}
	return tokens, &user, nil
}

// issueTokens generates an access + refresh JWT pair and stores the refresh token.
func (s *AuthService) issueTokens(ctx context.Context, userID, role string) (*AuthTokens, error) {
	access, err := s.tokens.GenerateAccessToken(userID, role)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh, err := s.tokens.GenerateRefreshToken(userID, role)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	uid, _ := uuid.Parse(userID)
	hash := pkgauth.HashToken(refresh)
	_, _ = s.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    uid,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(pkgauth.RefreshTokenTTL),
	})

	return &AuthTokens{AccessToken: access, RefreshToken: refresh}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func generateOTP(length int) (string, error) {
	digits := "0123456789"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[n.Int64()]
	}
	return string(result), nil
}

// ValidateAccessToken parses and validates a JWT token.
func (s *AuthService) ValidateAccessToken(token string) (*pkgauth.Claims, error) {
	return s.tokens.ValidateAccessToken(token)
}

func (s *AuthService) SendReviewStatusEmail(ctx context.Context, email, fullName, storeName, status string) error {
	if s.email == nil || s.cfg.SMTPUser == "" {
		return fmt.Errorf("email service not configured")
	}
	return s.email.SendSellerReviewNotification(email, fullName, storeName, status)
}

