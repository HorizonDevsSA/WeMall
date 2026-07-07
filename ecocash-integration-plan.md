# EcoCash Integration Plan — Marketplace Payment Service (Go)

## 1. Scope & Objectives

Integrate EcoCash Online Payments API into the marketplace payment service to support:

- **Payments** — buyer checkout via EcoCash wallet charge (`MER` transaction).
- **Refunds** — full/partial refund of a completed charge (`REF`).
- **Reversals** — cancel a pending/failed transaction before settlement (`REV`).
- **Payouts** — seller/vendor disbursements. *(Note: the provided EIP surface only exposes Charge, Lookup, and Refund/Reversal — there is no merchant-to-customer disbursement/B2C endpoint, and none is being pursued. Payouts are handled via a manual settlement bridge — see §7.)*

Target stack: Go, PostgreSQL, NATS (event publishing), Redis (idempotency/locking), clean architecture — consistent with the existing marketplace service conventions.

---

## 2. High-Level Architecture

```
                        ┌─────────────────────────┐
                        │   Marketplace API        │
                        │  (orders, chat, stores)  │
                        └───────────┬──────────────┘
                                    │ gRPC/HTTP
                        ┌───────────▼──────────────┐
                        │   payments-service (Go)   │
                        │                           │
                        │  ┌─────────────────────┐  │
                        │  │  domain (entities,   │  │
                        │  │  ports, use cases)   │  │
                        │  └─────────┬───────────┘  │
                        │  ┌─────────▼───────────┐  │
                        │  │  application layer    │  │
                        │  │  (charge, refund,     │  │
                        │  │   payout orchestrator)│  │
                        │  └─────────┬───────────┘  │
                        │  ┌─────────▼───────────┐  │
                        │  │  adapters:            │  │
                        │  │  - ecocash HTTP client │  │
                        │  │  - postgres repo       │  │
                        │  │  - nats producer      │  │
                        │  │  - webhook receiver    │  │
                        │  └─────────────────────┘  │
                        └───────────┬───────────────┘
                                    │
                     ┌──────────────┼───────────────┐
                     ▼              ▼               ▼
                PostgreSQL       NATS           EcoCash EIP
              (ledger, outbox)  (events)        (sandbox/prod)
```

Layering follows a standard clean-architecture split:

- `domain/` — `Transaction`, `Refund`, `Payout`, `Wallet` entities; port interfaces (`EcoCashGateway`, `TransactionRepository`, `EventPublisher`).
- `usecase/` — `ChargeCustomer`, `LookupTransaction`, `RefundTransaction`, `ReverseTransaction`, `DisbursePayout`.
- `adapter/ecocash/` — REST client implementing `EcoCashGateway`.
- `adapter/postgres/` — repository implementations.
- `adapter/nats/` — outbox-relay publisher.
- `adapter/http/` — inbound REST/webhook handlers.
- `cmd/` — service entrypoints (API server, outbox relay worker, webhook consumer).

---

## 3. EcoCash Gateway Client (`adapter/ecocash`)

### 3.1 Config

```go
type Config struct {
    BaseURL        string // https://developers.ecocash.co.zw/sandbox/payment/v1
    Username       string
    Password       string
    MerchantCode   string
    MerchantPin    string
    MerchantNumber string
    TerminalID     string
    CountryCode    string
    MerchantName   string
    SuperMerchant  string
    NotifyURL      string
    Timeout        time.Duration
}
```

Load via env vars / Vault/SSM; **never** log `MerchantPin` or the Basic Auth header. Separate config sets for sandbox vs production, selected by environment.

### 3.2 Client interface

```go
type Gateway interface {
    Charge(ctx context.Context, req ChargeRequest) (ChargeResponse, error)
    LookupTransaction(ctx context.Context, endUserID, correlator string) (LookupResponse, error)
    Refund(ctx context.Context, req RefundRequest) (RefundResponse, error)
}
```

- HTTP client: `net/http` with a bounded `http.Client{Timeout: ...}`, custom `RoundTripper` for:
  - Basic Auth header injection (`base64(username:password)`).
  - Structured logging (redacting PIN/credentials).
  - Retry with exponential backoff + jitter on `500`/`503`/network errors only (never retry `400`/`422` blindly — see §8).
- All requests/responses defined as explicit Go structs matching the documented JSON schema (`paymentAmount.charginginformation`, `chargeMetaData`, etc.) with `json` tags; avoid `map[string]interface{}` for anything that crosses a trust boundary.

### 3.3 `clientCorrelator` generation

- Generate as `<orderID>-<attemptSeq>` or a ULID stored against the internal transaction row *before* calling EcoCash, so retries reuse the same correlator (EcoCash treats a reused correlator as a lookup of the existing transaction — this is your idempotency guarantee).
- Persist correlator ↔ internal transaction ID mapping in Postgres with a unique constraint.

---

## 4. Data Model (PostgreSQL)

```sql
CREATE TABLE payment_transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id          UUID NOT NULL REFERENCES orders(id),
    client_correlator VARCHAR(64) NOT NULL UNIQUE,
    reference_code    VARCHAR(64) NOT NULL,
    tran_type         VARCHAR(8)  NOT NULL, -- MER, REF, REV
    end_user_id       VARCHAR(20) NOT NULL, -- encrypted at rest (pgcrypto) if treated as PII
    amount_cents      BIGINT NOT NULL,
    currency          VARCHAR(3) NOT NULL,
    status            VARCHAR(16) NOT NULL, -- PENDING, SUCCESS, FAILED, REFUNDED, REVERSED
    ecocash_status_code   VARCHAR(8),
    ecocash_status_msg    TEXT,
    ecocash_transaction_id VARCHAR(64),
    raw_request       JSONB,
    raw_response      JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_txn_order   ON payment_transactions(order_id);
CREATE INDEX idx_txn_status  ON payment_transactions(status);

CREATE TABLE refund_transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_txn_id   UUID NOT NULL REFERENCES payment_transactions(id),
    client_correlator VARCHAR(64) NOT NULL UNIQUE,
    amount_cents      BIGINT NOT NULL,
    status            VARCHAR(16) NOT NULL, -- PENDING, SUCCESS, FAILED
    ecocash_status_code VARCHAR(8),
    ecocash_status_msg  TEXT,
    raw_request       JSONB,
    raw_response      JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payouts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id         UUID NOT NULL REFERENCES sellers(id),
    amount_cents      BIGINT NOT NULL,
    currency          VARCHAR(3) NOT NULL,
    status            VARCHAR(16) NOT NULL, -- QUEUED, PROCESSING, PAID, FAILED, MANUAL_REVIEW
    method            VARCHAR(16) NOT NULL, -- manual_bulk (only method in use; enum left extensible for future adapters)
    provider_ref      VARCHAR(64),
    raw_response      JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Transactional outbox for NATS event publishing
CREATE TABLE outbox_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id  UUID NOT NULL,
    event_type    VARCHAR(64) NOT NULL, -- payment.succeeded, payment.failed, refund.completed, payout.paid
    payload       JSONB NOT NULL,
    published     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Notes:

- `end_user_id` (MSISDN) is PII — encrypt column-level (pgcrypto) or tokenize, matching the encrypted-PII pattern already used in the KYC service.
- Use the **transactional outbox pattern**: write the DB state change and outbox row in the same transaction, then a relay worker publishes to NATS and marks `published = true`. This guarantees no lost events even if NATS is briefly unavailable.

---

## 5. Payment Flow (Charge / `MER`)

1. Marketplace order service calls `payments-service.ChargeCustomer(orderID, msisdn, amount, currency)`.
2. Use case:
   - Validates MSISDN format (`263XXXXXXXXX` / `07XXXXXXXX` normalization), amount precision (2 dp), currency (`USD`/`ZWG` — confirmed both currencies share identical limits and PIN-matrix behavior in production, so no currency-specific branching is needed).
   - Generates `clientCorrelator`, inserts a `PENDING` row in `payment_transactions` (DB write happens **before** the outbound call, so a crash mid-call is recoverable via reconciliation, §9).
   - Calls `Gateway.Charge()` with `tranType=MER`, `notifyUrl` pointed at the service's webhook endpoint, merchant fields from config.
3. Since **all sandbox responses return HTTP 200**, the use case must branch on `statusMessage`/`statusCode`, not on HTTP status alone:
   - `Transaction Successful` → mark row `SUCCESS` (or `PENDING` if EcoCash's real production flow confirms asynchronously via webhook — treat sandbox `PENDING` as authoritative until a webhook/lookup confirms).
   - `Insufficient Balance`, `Transaction Failed - Invalid PIN`, `Transaction Limit Exceeded` → mark `FAILED`, store reason.
4. Emit `outbox_events` row (`payment.succeeded` / `payment.failed`) in the same DB transaction as the status update.
5. Return a normalized result (`OrderPaymentResult{Status, FailureReason}`) to the marketplace order flow — never leak raw EcoCash payloads to the client-facing API.
6. **Webhook (`notifyUrl`) handler** — separate endpoint, verifies the payload is for a known `client_correlator`, updates status idempotently (ignore if already terminal), republishes event if state changed. Treat this as the source of truth for any transaction that started `PENDING`.
7. **SMS notification** is EcoCash-side and informational only — do not depend on it for business logic.

---

## 6. Refund & Reversal Flow (`REF` / `REV`)

- **Reversal (`REV`)**: used when a transaction is still `PENDING`/unsettled (e.g., customer never completed USSD PIN, or failed shortly after). Cancels before settlement.
- **Refund (`REF`)**: used against a `SUCCESS` transaction, full or partial.

Use case `RefundTransaction(originalTxnID, amount, reason)`:

1. Load original transaction; verify `status == SUCCESS` and `amount <= original.amount - already_refunded`.
2. Reject locally (no API call) for:
   - Amount exceeding original (mirrors `E012`).
   - Transaction not eligible (already refunded / reversed) — mirrors `E009`.
3. Generate a new `clientCorrelator` for the refund record, insert `PENDING` row in `refund_transactions`.
4. Call `Gateway.Refund()` with `tranType=REF` (or `REV` if pre-settlement), referencing the original `referenceCode`/`transactionId`.
5. On success: update `refund_transactions.status = SUCCESS`, update parent `payment_transactions.status = REFUNDED` (or `PARTIALLY_REFUNDED` if you extend the model), emit `refund.completed` outbox event so order/seller-ledger services can react (e.g., release/claw back escrowed seller balance).
6. On EcoCash error codes:
   - `E009` (not eligible), `E012` (exceeds original) → surface as domain validation errors, no retry.
   - `E010`/`E011`/`E013` → surface to support/ops queue, no automatic retry.
   - `E014`/`E015` → retry with backoff, then move to `MANUAL_REVIEW` if retries exhausted.

---

## 7. Payouts (Seller Disbursement) — Gap Analysis & Plan

**Confirmed:** the documented EIP surface (Charge, Lookup, Refund/Reversal) is a **collections API only** — it moves money *from* a customer wallet *to* the merchant. There is **no B2C/bulk-disbursement API available**; the decision is to work with what this API surface provides rather than wait on or pursue a separate disbursement product.

Approach:

1. **Manual settlement bridge** (the permanent approach, not an interim one): model payouts as a `payouts` table with `status=QUEUED`, and run seller settlement through a manual/batched bulk-transfer process (finance team executes via the EcoCash merchant portal or bank rail), with the payments-service only tracking the payout request, amount, and reconciliation status — no automated wallet push.
   - Mark `method = manual_bulk` so it's auditable.
2. **Design the domain layer payout-agnostic** anyway: `PayoutGateway` interface with a single method `Disburse(ctx, PayoutRequest) (PayoutResult, error)`, so that *if* an automated disbursement path becomes available later, swapping in a new adapter requires no change to use cases — consistent with the ports-and-adapters approach used elsewhere in the marketplace. This costs little now and avoids a rewrite if the situation changes.
3. Expose payout status to sellers as `QUEUED` / `PROCESSING` / `PAID` / `FAILED`, sourced from whatever settlement process finance runs, ingested via a small internal reconciliation import (CSV/API) into the `payouts` table.
4. Since payouts are manual, build in reasonable operational safeguards: a finance-facing report/export of `QUEUED` payouts, an audit trail on status transitions, and reconciliation against the merchant's actual EcoCash/bank balance before marking `PAID`.

Note: the go-live test script (§10) has no test cases for payouts, consistent with payouts being outside this API's scope.

---

## 8. Error Handling Strategy

| Category | Codes | Handling |
|---|---|---|
| Client/validation errors | E001–E005 | Fail fast, no retry, return 4xx to caller with actionable message |
| Auth/config errors | E006, E007 | Fail fast, alert on-call (should never reach production if config validated at startup) |
| Business rule errors | E009, E011, E012, E013 | No retry; surface as domain-level rejection; some (E011, E013) may warrant a user-facing "try another payment method" message |
| Funds errors | E010 | No retry; user-facing "insufficient balance" |
| Not found | E008 | No retry; likely a bug/race — log at WARN, investigate |
| Transient errors | E014, E015 | Retry with exponential backoff (e.g., 1s, 2s, 4s, cap 30s, max 5 attempts) then move to `MANUAL_REVIEW` / dead-letter |
| Network timeout / connection errors | n/a | Treat as transient — but since a charge may have actually succeeded on EcoCash's side despite a client-side timeout, **always reconcile via Transaction Lookup before retrying a charge**, never blindly resubmit a new charge for the same order |

Golden rule for `MER` charges specifically: **never retry a charge with a new correlator after a timeout** — first call `LookupTransaction` with the original correlator to check if it actually went through, to avoid double-charging the customer.

**Charge response SLA:** allow up to **60 seconds** for a charge response (covering the customer's USSD PIN prompt round-trip) before treating the call as timed out client-side. Set the HTTP client timeout on `Gateway.Charge()` to 60s, and only fall back to `LookupTransaction` polling once that window has elapsed without a response.

---

## 9. Idempotency & Reconciliation

- **Idempotency key = `client_correlator`**, enforced by a unique DB constraint plus reuse-on-retry logic in the use case layer.
- **Reconciliation worker** (scheduled job, e.g., every 5 min): scans `payment_transactions`/`refund_transactions` stuck in `PENDING` older than **60 seconds** (the confirmed charge response SLA, §8), calls `LookupTransaction`/status check, and resolves them — covers webhook delivery failures and crashed in-flight requests.
- NATS `outbox_events` relay is itself idempotent (publish-then-mark-published, safe to retry).

---

## 10. Testing Plan

- **Unit tests**: use case layer with a mocked `Gateway` interface (table-driven tests covering each PIN scenario's response shape).
- **Sandbox integration tests**: automated suite driving the actual sandbox using the **PIN Test Matrix**:

  | PIN | Scenario | Expect |
  |---|---|---|
  | 0000 | Success | `payment_transactions.status = SUCCESS`, `payment.succeeded` event emitted |
  | 1111 | Insufficient funds | `status = FAILED`, reason recorded |
  | 2222 | Invalid PIN | `status = FAILED`, reason recorded |
  | 9999 | Limit exceeded | `status = FAILED`, reason recorded |

- Also cover: transaction lookup after each scenario, refund of a `SUCCESS` transaction, refund-exceeds-original rejection (E012), reversal of a still-pending transaction, duplicate-correlator behavior (E005 / correlator reuse), webhook replay/idempotency.
- **Test numbers**: whitelist and OTP-verify at least one real sandbox MSISDN per environment (dev, CI) via the Test Numbers flow; store as a secret, not hardcoded.
- **Go-live requirement**: populate and pass the EcoCash-provided Test Script (Excel) — TC-001 through TC-006 — attach to the production access request. Track this as an explicit release checklist item; note it currently has **no payout test case** (see §7).

---

## 11. Security Considerations

- Basic Auth credentials and `merchantPin` stored in a secrets manager, injected via env at runtime — never in source control or logs.
- TLS enforced for all calls (sandbox/prod are HTTPS already).
- MSISDN treated as PII: encrypt at rest, mask in logs (`773***653`), restrict access via role-based DB permissions.
- Webhook endpoint: **`notifyUrl` payloads are unsigned** (confirmed) — do not treat the source as trusted by default. Mitigate with: an unguessable, per-environment webhook path/token embedded in the `notifyUrl` itself, an IP allowlist for EcoCash's known egress ranges if available, and strict payload validation (the callback must match an existing `client_correlator` already in `PENDING` state — reject anything else). Treat the webhook purely as a *trigger* to re-confirm status via `LookupTransaction` rather than as authoritative on its own, since it cannot be cryptographically verified.
- Rate-limit and circuit-break the EcoCash client to avoid cascading failures during EcoCash-side incidents (`E015`).

---

## 12. Proposed Package Layout

```
payments-service/
├── cmd/
│   ├── api/                 # HTTP/gRPC server
│   ├── outbox-relay/        # NATS publisher worker
│   └── reconciler/          # scheduled pending-txn resolver
├── internal/
│   ├── domain/
│   │   ├── payment.go
│   │   ├── refund.go
│   │   ├── payout.go
│   │   └── ports.go
│   ├── usecase/
│   │   ├── charge_customer.go
│   │   ├── lookup_transaction.go
│   │   ├── refund_transaction.go
│   │   ├── reverse_transaction.go
│   │   └── disburse_payout.go
│   ├── adapter/
│   │   ├── ecocash/
│   │   │   ├── client.go
│   │   │   ├── types.go
│   │   │   └── errors.go
│   │   ├── postgres/
│   │   ├── nats/
│   │   └── httpapi/
│   │       ├── handlers.go
│   │       └── webhook.go
│   └── config/
├── migrations/
└── go.mod
```

---

## 13. Rollout Milestones

1. **Week 1** — Domain models, Postgres migrations, `EcoCashGateway` client with unit tests against mocked HTTP responses.
2. **Week 2** — Charge flow end-to-end against sandbox (all 4 PIN scenarios), webhook receiver, outbox → NATS relay.
3. **Week 3** — Refund/reversal flow, reconciliation worker, error-handling matrix implemented per §8.
4. **Week 4** — Build out the manual payout bridge (§7) and finance reconciliation workflow, load/chaos testing (timeouts, duplicate correlators), security review.
5. **Week 5** — Complete EcoCash Test Script, submit production access request, staged rollout behind a feature flag with a low transaction-value cap initially.

