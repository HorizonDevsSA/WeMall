package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const (
	// maxRetries for transient errors (E014/E015 and network errors).
	maxRetries = 5
	// baseBackoff is the initial retry wait.
	baseBackoff = 1 * time.Second
	// maxBackoff caps the retry wait to prevent runaway sleeping.
	maxBackoff = 30 * time.Second
	// chargeTimeout is the full window for a charge response, covering the
	// customer's USSD PIN prompt round-trip (§8 of integration plan).
	chargeTimeout = 60 * time.Second
	// defaultTimeout for lookup and refund calls.
	defaultTimeout = 30 * time.Second
)

// GatewayConfig holds the EcoCash EIP connection parameters.
type GatewayConfig struct {
	BaseURL        string
	Username       string
	Password       string // Basic Auth — never log
	MerchantCode   string
	MerchantPin    string // never log
	MerchantNumber string
	TerminalID     string
	CountryCode    string
	MerchantName   string
	SuperMerchant  string
	NotifyURL      string // e.g. https://api.wemall.co.zw/webhooks/ecocash
	ProxySecret    string
}

// Gateway is the port interface consumed by the use-case layer.
type Gateway interface {
	Charge(ctx context.Context, req ChargeRequest) (ChargeResponse, error)
	LookupTransaction(ctx context.Context, endUserID, correlator string) (LookupResponse, error)
	Refund(ctx context.Context, req RefundRequest) (ChargeResponse, error)
}

// Client implements Gateway against the EcoCash EIP REST API.
type Client struct {
	cfg        GatewayConfig
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewClient constructs a ready-to-use EcoCash gateway client.
func NewClient(cfg GatewayConfig, logger zerolog.Logger) *Client {
	return &Client{
		cfg: cfg,
		// Charge calls need the full 60 s USSD window; all other calls use a
		// separate shorter client constructed inline (see do()).
		httpClient: &http.Client{Timeout: chargeTimeout},
		logger:     logger,
	}
}

// ── Public Methods ────────────────────────────────────────────────────────────

// Charge sends a MER transaction to EcoCash and returns the parsed response.
// Retries are only attempted on E014/E015 or network-level errors.
func (c *Client) Charge(ctx context.Context, req ChargeRequest) (ChargeResponse, error) {
	req.Merchant = c.merchantInfo(req.Amount.ChargingInformation.Currency)
	req.NotifyURL = c.cfg.NotifyURL

	return c.doCharge(ctx, req, "transactions")
}

// Refund sends a REF or REV transaction to EcoCash.
func (c *Client) Refund(ctx context.Context, req RefundRequest) (ChargeResponse, error) {
	req.Merchant = c.merchantInfo(req.Amount.ChargingInformation.Currency)
	req.NotifyURL = c.cfg.NotifyURL

	return c.doRefund(ctx, req)
}

// LookupTransaction calls the EcoCash lookup endpoint to resolve a transaction's
// current status. This must be called before retrying any charge after a
// timeout, to avoid double-charging the customer.
func (c *Client) LookupTransaction(ctx context.Context, endUserID, correlator string) (LookupResponse, error) {
	path := fmt.Sprintf("transactions/%s/%s", endUserID, correlator)

	httpClient := &http.Client{Timeout: defaultTimeout}
	body, err := c.get(ctx, httpClient, path)
	if err != nil {
		return LookupResponse{}, fmt.Errorf("ecocash lookup: %w", err)
	}

	var resp LookupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return LookupResponse{}, fmt.Errorf("ecocash lookup decode: %w", err)
	}
	return resp, nil
}

// ── Internal Helpers ──────────────────────────────────────────────────────────

func (c *Client) doCharge(ctx context.Context, req ChargeRequest, path string) (ChargeResponse, error) {
	attempt := 0
	for {
		attempt++
		resp, err := c.postCharge(ctx, c.httpClient, path, req)
		if err == nil {
			return resp, nil
		}

		// Only retry transient errors; bail immediately on anything else.
		if !IsRetryable(err) {
			return ChargeResponse{}, err
		}
		if attempt >= maxRetries {
			return ChargeResponse{}, fmt.Errorf("ecocash charge: max retries reached: %w", err)
		}

		wait := backoff(attempt)
		c.logger.Warn().Err(err).Int("attempt", attempt).Dur("wait", wait).Msg("retrying ecocash charge after transient error")
		select {
		case <-ctx.Done():
			return ChargeResponse{}, ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (c *Client) doRefund(ctx context.Context, req RefundRequest) (ChargeResponse, error) {
	path := "transactions/refund"
	httpClient := &http.Client{Timeout: defaultTimeout}

	attempt := 0
	for {
		attempt++
		resp, err := c.postRefund(ctx, httpClient, path, req)
		if err == nil {
			return resp, nil
		}
		if !IsRetryable(err) {
			return ChargeResponse{}, err
		}
		if attempt >= maxRetries {
			return ChargeResponse{}, fmt.Errorf("ecocash refund: max retries reached: %w", err)
		}

		wait := backoff(attempt)
		c.logger.Warn().Err(err).Int("attempt", attempt).Dur("wait", wait).Msg("retrying ecocash refund after transient error")
		select {
		case <-ctx.Done():
			return ChargeResponse{}, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// postCharge serialises req, calls the EcoCash endpoint, and parses the response.
func (c *Client) postCharge(ctx context.Context, hc *http.Client, path string, req ChargeRequest) (ChargeResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return ChargeResponse{}, fmt.Errorf("marshal charge request: %w", err)
	}

	rawResp, err := c.post(ctx, hc, path, bodyBytes)
	if err != nil {
		return ChargeResponse{}, err
	}

	var resp ChargeResponse
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		c.logger.Error().Err(err).Str("body", string(rawResp)).Msg("Failed to decode charge response")
		return ChargeResponse{}, fmt.Errorf("decode charge response: %w", err)
	}

	if gatewayErr := ParseEcoCashError(resp.StatusCode, resp.StatusMessage); gatewayErr != nil {
		return resp, gatewayErr
	}
	return resp, nil
}

// postRefund is identical to postCharge but for RefundRequest.
func (c *Client) postRefund(ctx context.Context, hc *http.Client, path string, req RefundRequest) (ChargeResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return ChargeResponse{}, fmt.Errorf("marshal refund request: %w", err)
	}

	rawResp, err := c.post(ctx, hc, path, bodyBytes)
	if err != nil {
		return ChargeResponse{}, err
	}

	var resp ChargeResponse
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		return ChargeResponse{}, fmt.Errorf("decode refund response: %w", err)
	}

	if gatewayErr := ParseEcoCashError(resp.StatusCode, resp.StatusMessage); gatewayErr != nil {
		return resp, gatewayErr
	}
	return resp, nil
}

// post executes an authenticated POST request to the EcoCash EIP.
func (c *Client) post(ctx context.Context, hc *http.Client, path string, body []byte) ([]byte, error) {
	url := c.cfg.BaseURL + "/" + path

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create post request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+c.basicAuth())
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if c.cfg.ProxySecret != "" {
		httpReq.Header.Set("X-WeMall-Proxy-Secret", c.cfg.ProxySecret)
	}

	c.logger.Debug().
		Str("method", "POST").
		Str("url", url).
		// Never log the body as it may contain merchantPin.
		Msg("ecocash outbound request")

	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ecocash http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ecocash response body: %w", err)
	}

	// EcoCash always returns 200 — non-200 indicates a proxy/infra error.
	if resp.StatusCode >= 500 {
		return nil, ErrServiceUnavailable
	}

	c.logger.Debug().
		Str("url", url).
		Int("http_status", resp.StatusCode).
		Msg("ecocash response received")

	return respBody, nil
}

// get executes an authenticated GET request to the EcoCash EIP.
func (c *Client) get(ctx context.Context, hc *http.Client, path string) ([]byte, error) {
	url := c.cfg.BaseURL + "/" + path

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create get request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Basic "+c.basicAuth())
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if c.cfg.ProxySecret != "" {
		httpReq.Header.Set("X-WeMall-Proxy-Secret", c.cfg.ProxySecret)
	}

	c.logger.Debug().
		Str("method", "GET").
		Str("url", url).
		Msg("ecocash outbound request")

	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ecocash http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ecocash response body: %w", err)
	}

	if resp.StatusCode >= 500 {
		return nil, ErrServiceUnavailable
	}

	return body, nil
}

// basicAuth returns the Base64-encoded "username:password" credential string.
// Never pass this to a logger.
func (c *Client) basicAuth() string {
	cred := c.cfg.Username + ":" + c.cfg.Password
	return base64.StdEncoding.EncodeToString([]byte(cred))
}

// merchantInfo assembles the MerchantInfo block for a request.
func (c *Client) merchantInfo(currency string) MerchantInfo {
	return MerchantInfo{
		MerchantCode:   c.cfg.MerchantCode,
		MerchantPin:    c.cfg.MerchantPin, // redacted in logs by not logging requests
		MerchantNumber: c.cfg.MerchantNumber,
		TerminalID:     c.cfg.TerminalID,
		Location:       "WeMall Online",
		Currency:       currency,
		CountryCode:    c.cfg.CountryCode,
		MerchantName:   c.cfg.MerchantName,
		SuperMerchant:  c.cfg.SuperMerchant,
	}
}

// backoff returns the exponential back-off duration for the given attempt,
// capped at maxBackoff, with ±20% jitter.
func backoff(attempt int) time.Duration {
	exp := math.Pow(2, float64(attempt-1))
	base := float64(baseBackoff) * exp
	jitter := (rand.Float64()*0.4 - 0.2) * base // ±20%
	d := time.Duration(base + jitter)
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}
