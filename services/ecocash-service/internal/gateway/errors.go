package gateway

import (
	"errors"
	"strings"
)

// EcoCash documented error codes. See integration plan §8 for handling matrix.
var (
	// ── Client / validation errors (no retry) ──────────────────────────────
	ErrInvalidRequest       = errors.New("E001: Invalid Request Format")
	ErrMissingField         = errors.New("E002: Missing Required Field")
	ErrInvalidAmount        = errors.New("E003: Invalid Amount Format")
	ErrCurrencyMismatch     = errors.New("E004: Currency Mismatch")
	ErrDuplicateCorrelator  = errors.New("E005: Duplicate Client Correlator")

	// ── Auth / config errors (should never reach production) ───────────────
	ErrAuthFailed           = errors.New("E006: Authentication Failed")
	ErrMerchantNotAuthorized= errors.New("E007: Merchant Not Authorized")

	// ── Business rule errors (no retry) ────────────────────────────────────
	ErrTransactionNotFound  = errors.New("E008: Original Transaction Not Found")
	ErrNotEligible          = errors.New("E009: Transaction Not Eligible for Refund/Reversal")
	ErrInsufficientFunds    = errors.New("E010: Insufficient Customer Wallet Balance")
	ErrRefundNotAllowed     = errors.New("E011: Refund Period Expired")
	ErrRefundExceedsOriginal= errors.New("E012: Refund Amount Exceeds Original Transaction")
	ErrReversalNotAllowed   = errors.New("E013: Reversal Not Allowed After Settlement")

	// ── Transient errors (retry after backoff) ─────────────────────────────
	ErrSystemTemporary      = errors.New("E014: Temporary System Error")
	ErrServiceUnavailable   = errors.New("E015: Service Unavailable")
)

// ParseEcoCashError maps an EcoCash statusCode to a known error constant.
// Returns nil if statusCode indicates success, or an unknown generic error if
// the code is not documented.
func ParseEcoCashError(statusCode, statusMessage string) error {
	if isSuccessCode(statusCode) {
		return nil
	}

	switch statusCode {
	case "E001":
		return ErrInvalidRequest
	case "E002":
		return ErrMissingField
	case "E003":
		return ErrInvalidAmount
	case "E004":
		return ErrCurrencyMismatch
	case "E005":
		return ErrDuplicateCorrelator
	case "E006":
		return ErrAuthFailed
	case "E007":
		return ErrMerchantNotAuthorized
	case "E008":
		return ErrTransactionNotFound
	case "E009":
		return ErrNotEligible
	case "E010":
		return ErrInsufficientFunds
	case "E011":
		return ErrRefundNotAllowed
	case "E012":
		return ErrRefundExceedsOriginal
	case "E013":
		return ErrReversalNotAllowed
	case "E014":
		return ErrSystemTemporary
	case "E015":
		return ErrServiceUnavailable
	default:
		// Unknown error code — wrap the message for observability
		return errors.New("EcoCash error: " + statusCode + " — " + statusMessage)
	}
}

// isSuccessCode checks if the EcoCash statusCode / statusMessage indicates
// success. In sandbox all responses are HTTP 200, so we must inspect the body.
// Success indicators: "Transaction Successful", "Pending", or no error code.
func isSuccessCode(statusCode string) bool {
	if statusCode == "" || statusCode == "0" {
		return true
	}
	// EcoCash may return a string like "SUCCESS" or "Transaction Successful"
	// in the statusMessage field — lenient check:
	lower := strings.ToLower(statusCode)
	return lower == "success" || lower == "ok"
}

// IsRetryable checks if the error is transient and safe to retry with backoff.
func IsRetryable(err error) bool {
	return errors.Is(err, ErrSystemTemporary) || errors.Is(err, ErrServiceUnavailable)
}
