package gateway

// ChargeRequest is the payload sent to EcoCash for a MER transaction.
type ChargeRequest struct {
	ClientCorrelator string `json:"clientCorrelator"`
	NotifyURL        string `json:"notifyUrl"`
	ReferenceCode    string `json:"referenceCode"`
	TranType         string `json:"tranType"` // MER
	EndUserID        string `json:"endUserId"` // normalised MSISDN: 2637XXXXXXXX
	Remark           string `json:"remark"`
	Merchant         MerchantInfo `json:"chargeMetaData"`
	Amount           PaymentAmount `json:"paymentAmount"`
}

// RefundRequest is the payload sent for REF or REV transactions.
type RefundRequest struct {
	ClientCorrelator    string `json:"clientCorrelator"`
	NotifyURL           string `json:"notifyUrl"`
	ReferenceCode       string `json:"referenceCode"`
	TranType            string `json:"tranType"` // REF or REV
	OriginalTxnID       string `json:"originalTransactionId,omitempty"`
	OriginalReference   string `json:"originalReferenceCode,omitempty"`
	EndUserID           string `json:"endUserId"`
	Merchant            MerchantInfo `json:"chargeMetaData"`
	Amount              PaymentAmount `json:"paymentAmount"`
}

// MerchantInfo holds the merchant identification fields required by EcoCash.
type MerchantInfo struct {
	MerchantCode   string `json:"merchantCode"`
	MerchantPin    string `json:"merchantPin"`  // redacted from logs
	MerchantNumber string `json:"merchantNumber"`
	TerminalID     string `json:"terminalID"`
	Location       string `json:"location"`
	Currency       string `json:"currency"`
	CountryCode    string `json:"countryCode"`
	MerchantName   string `json:"merchantName"`
	SuperMerchant  string `json:"superMerchantName,omitempty"`
}

// PaymentAmount mirrors the EcoCash paymentAmount object.
type PaymentAmount struct {
	ChargingInformation ChargingInformation `json:"charginginformation"`
	ChargingMetadata    ChargingMetadata    `json:"charginginformationbeforetax"`
}

// ChargingInformation holds the amount and currency for the charge.
type ChargingInformation struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

// ChargingMetadata mirrors the pre-tax charging information field.
type ChargingMetadata struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

// ChargeResponse is the deserialized EcoCash charge/refund API response.
// All EcoCash responses come back as HTTP 200 — success/failure is determined
// by inspecting statusMessage / statusCode.
type ChargeResponse struct {
	ClientCorrelator    string `json:"clientCorrelator"`
	TransactionID       string `json:"transactionId"`
	ReferenceCode       string `json:"referenceCode"`
	StatusCode          string `json:"statusCode"`
	StatusMessage       string `json:"statusMessage"`
	Description         string `json:"description"`
	EndUserID           string `json:"endUserId"`
	PaymentAmount       PaymentAmount `json:"paymentAmount"`
}

// LookupResponse is the deserialized EcoCash transaction-lookup response.
type LookupResponse struct {
	ClientCorrelator string `json:"clientCorrelator"`
	TransactionID    string `json:"transactionId"`
	StatusCode       string `json:"statusCode"`
	StatusMessage    string `json:"statusMessage"`
	Description      string `json:"description"`
	TranType         string `json:"tranType"`
}
