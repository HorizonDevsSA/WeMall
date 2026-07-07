package service

import (
	"fmt"
	"regexp"
	"strings"
)

// E.164 Zimbabwean MSISDN: 2637XXXXXXXX or 2638XXXXXXXX (12 digits total).
var e164ZW = regexp.MustCompile(`^2637\d{8}$|^2638\d{8}$`)

// NormalizeMSISDN converts Zimbabwean local/short formats to E.164 (263...).
// Accepted inputs: 07XXXXXXXX, 08XXXXXXXX, 7XXXXXXXX, 8XXXXXXXX, 2637XXXXXXXX.
// Returns an error if the number cannot be normalised.
func NormalizeMSISDN(raw string) (string, error) {
	n := strings.TrimSpace(raw)
	n = strings.ReplaceAll(n, " ", "")
	n = strings.ReplaceAll(n, "-", "")

	switch {
	case strings.HasPrefix(n, "2637") || strings.HasPrefix(n, "2638"):
		// Already E.164
	case strings.HasPrefix(n, "07"):
		n = "263" + n[1:]
	case strings.HasPrefix(n, "08"):
		n = "263" + n[1:]
	case strings.HasPrefix(n, "7") && len(n) == 9:
		n = "2637" + n[1:]
	case strings.HasPrefix(n, "8") && len(n) == 9:
		n = "2638" + n[1:]
	}

	if !e164ZW.MatchString(n) {
		return "", fmt.Errorf("invalid Zimbabwean MSISDN: %s", raw)
	}
	return n, nil
}

// MaskMSISDN returns a masked representation for logging/storage.
// e.g. 2637731234567 → 263773***567
func MaskMSISDN(normalized string) string {
	if len(normalized) < 7 {
		return "***"
	}
	prefix := normalized[:7]  // 2637731
	suffix := normalized[len(normalized)-3:]
	return prefix + "***" + suffix
}
