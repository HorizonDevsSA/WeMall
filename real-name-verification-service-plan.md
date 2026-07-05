# Real-Name Verification Service — Zimbabwe Market
### Technical Design & Implementation Plan (Go)

**Owner:** Backend Engineering
**Status:** Draft v1
**Scope:** KYC / identity verification microservice for Passport, National ID, and Driver's Licence, targeting Zimbabwean users.

---

## 1. Goals & Non-Goals

### Goals
- Verify that a user is a real, uniquely-identifiable person before granting privileged access (e.g. selling on the platform, high-value transactions, payouts).
- Support the three ID document types commonly held in Zimbabwe:
  - **National ID** (format: `63-XXXXXXX-X-XX` — district code, serial, check letter, issuing district)
  - **Passport** (ZWE MRZ passport, ICAO 9303 format)
  - **Driver's Licence** (Zimbabwe VID format)
- Extract structured data from document images via OCR.
- Perform liveness/selfie-to-document face match (optional phase 2).
- Produce a durable, auditable verification record with a status and confidence score.
- Be provider-agnostic for OCR/face-match so we can swap vendors without a rewrite.

### Non-Goals (v1)
- Building our own OCR/ML models from scratch (we integrate a provider first, evaluate in-house later).
- Storing raw document images ourselves — this is delegated to the existing **Media Service**, which returns URLs. We only ever store the URL + extracted metadata + hashes.
- Real-time integration with Zimbabwe's national ID registry (ZRP/Registrar-General) — not publicly available; flagged as a future risk/dependency.

---

## 2. High-Level Architecture

```
                     ┌───────────────────────┐
                     │      Client App        │
                     └──────────┬─────────────┘
                                │ 1. Upload doc images
                                ▼
                     ┌───────────────────────┐
                     │     Media Service       │  (existing, out of scope)
                     │  stores file, returns   │
                     │  signed URL + media_id  │
                     └──────────┬─────────────┘
                                │ 2. media_id + url
                                ▼
          ┌────────────────────────────────────────────┐
          │        Identity Verification Service         │  ← this plan
          │                (Go, modular monolith)         │
          │                                              │
          │  ┌────────────┐  ┌────────────┐  ┌─────────┐ │
          │  │  API Layer  │→│ Verification │→│  OCR     │ │
          │  │ (REST/gRPC) │  │ Orchestrator│  │ Provider │ │
          │  └────────────┘  └─────┬───────┘  │ Adapter  │ │
          │                        │          └─────────┘ │
          │                        ▼                       │
          │                ┌───────────────┐               │
          │                │ Rules Engine /  │              │
          │                │ Validators      │              │
          │                │ (per doc type)  │              │
          │                └───────┬─────────┘              │
          │                        ▼                        │
          │                ┌───────────────┐                │
          │                │  Face Match     │ (phase 2)     │
          │                │  Adapter        │               │
          │                └───────┬─────────┘               │
          │                        ▼                          │
          │                ┌───────────────┐                  │
          │                │ Postgres:       │                 │
          │                │ verifications,  │                 │
          │                │ extracted_fields│                 │
          │                └───────┬─────────┘                 │
          │                        ▼                            │
          │                ┌───────────────┐                    │
          │                │  Event Bus      │  → notifies       │
          │                │ (Kafka/NATS)    │    other services │
          │                └───────────────┘                    │
          └────────────────────────────────────────────┘
                                │ 3. status webhook/event
                                ▼
                     ┌───────────────────────┐
                     │  Downstream Services    │
                     │ (User/Store/Payments)   │
                     └───────────────────────┘
```

**Deployment shape:** single Go service to start (modular monolith, clean architecture layers), split into `ocr-worker` as a separate deployable once volume justifies it (OCR is CPU/GPU heavy and async-friendly).

---

## 3. Domain Model

### 3.1 Core Entities

```go
type DocumentType string

const (
    DocTypeNationalID     DocumentType = "national_id"
    DocTypePassport       DocumentType = "passport"
    DocTypeDriversLicence DocumentType = "drivers_licence"
)

type VerificationStatus string

const (
    StatusPending          VerificationStatus = "pending"
    StatusProcessing       VerificationStatus = "processing"
    StatusNeedsReview      VerificationStatus = "needs_review"
    StatusApproved         VerificationStatus = "approved"
    StatusRejected         VerificationStatus = "rejected"
    StatusExpired          VerificationStatus = "expired"
)

type Verification struct {
    ID              uuid.UUID
    UserID          string
    DocumentType    DocumentType
    FrontMediaID    string          // reference to Media Service, not the file itself
    FrontMediaURL   string
    BackMediaID     *string
    BackMediaURL    *string
    SelfieMediaID   *string         // for face match, phase 2
    Status          VerificationStatus
    Confidence      float64         // 0.0 - 1.0 aggregate score
    RejectionReason *string
    ReviewedBy      *string         // manual reviewer, if escalated
    CreatedAt       time.Time
    UpdatedAt       time.Time
    ExpiresAt       *time.Time      // re-verification cadence
}

// ExtractedFields is document-type-specific; stored as JSONB with a typed
// view per document type for querying/validation.
type ExtractedFields struct {
    VerificationID uuid.UUID
    FullName       string
    DateOfBirth    *time.Time
    Sex            *string
    DocumentNumber string          // national ID number / passport no / licence no
    Nationality    *string
    IssueDate      *time.Time
    ExpiryDate     *time.Time
    IssuingAuthority *string
    RawOCRPayload  json.RawMessage // full provider response, for audit/debug
    OCRConfidence  float64
}
```

### 3.2 Postgres Tables (sketch)

```sql
CREATE TABLE verifications (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           TEXT NOT NULL,
    document_type     TEXT NOT NULL,
    front_media_id    TEXT NOT NULL,
    front_media_url   TEXT NOT NULL,
    back_media_id     TEXT,
    back_media_url    TEXT,
    selfie_media_id   TEXT,
    status            TEXT NOT NULL DEFAULT 'pending',
    confidence        NUMERIC(4,3),
    rejection_reason  TEXT,
    reviewed_by       TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ
);

CREATE TABLE extracted_fields (
    verification_id   UUID PRIMARY KEY REFERENCES verifications(id),
    full_name         TEXT,
    date_of_birth     DATE,
    sex               TEXT,
    document_number   TEXT NOT NULL,
    nationality       TEXT,
    issue_date        DATE,
    expiry_date       DATE,
    issuing_authority TEXT,
    raw_ocr_payload   JSONB,
    ocr_confidence    NUMERIC(4,3)
);

-- Prevent the same physical document being used across many accounts
CREATE UNIQUE INDEX idx_document_number_type
    ON extracted_fields (document_number)
    WHERE document_number IS NOT NULL;

CREATE INDEX idx_verifications_user_id ON verifications(user_id);
CREATE INDEX idx_verifications_status ON verifications(status);
```

> Note: document numbers are PII. Consider storing a keyed HMAC hash for the uniqueness index and encrypting the plaintext value at rest (see Security section).

---

## 4. Zimbabwe Document Specifics

| Document | Key fields to extract | Format notes | Validation |
|---|---|---|---|
| **National ID** | ID number, full name, DOB, sex, district | `NN-NNNNNNN-X-NN` (2-digit district, 7-digit serial, check letter, 2-digit district again) | Regex + check-letter algorithm validation |
| **Passport** | Full name, passport no., DOB, sex, nationality, expiry, MRZ lines | ICAO 9303 2-line MRZ on photo page | MRZ checksum validation (composite check digit) |
| **Drivers Licence** | Licence no., full name, DOB, classes, expiry | VID card, front/back | Expiry date must be in the future; class codes validated against known set |

**Validators module** (`internal/validators`) implements one `Validator` per document type:

```go
type Validator interface {
    Validate(fields ExtractedFields) (ValidationResult, error)
}

type ValidationResult struct {
    Valid      bool
    Confidence float64
    Issues     []string // e.g. "checksum_failed", "expired", "dob_mismatch"
}
```

National ID checksum and passport MRZ checksum are pure functions we implement ourselves (no external dependency needed) — this is the single most valuable Zimbabwe-specific piece of logic in the whole service, since a generic OCR vendor won't validate local check-digit schemes.

---

## 5. OCR Strategy

### 5.1 Provider abstraction

Define an interface so the OCR vendor is swappable and testable:

```go
type OCRProvider interface {
    ExtractDocument(ctx context.Context, req OCRRequest) (OCRResult, error)
}

type OCRRequest struct {
    DocumentType DocumentType
    FrontImageURL string
    BackImageURL  *string
}

type OCRResult struct {
    Fields     ExtractedFields
    Confidence float64
    Provider   string
    RawResponse json.RawMessage
}
```

### 5.2 Candidate providers (evaluate in this order)

1. **AWS Textract (AnalyzeID)** — purpose-built for ID documents, handles MRZ, good baseline accuracy, pay-per-call, no infra to run.
2. **Google Document AI (Identity Document Proofing)** — comparable, good MRZ support.
3. **Regional/African ID specialists** (e.g. Smile Identity, Youverify) — some have explicit Zimbabwe National ID support and combine OCR + liveness + face match in one call; worth a vendor evaluation since general-purpose OCR vendors may under-perform on the local ID layout.
4. **Self-hosted fallback** (Tesseract + custom field-position templates, or PaddleOCR) — higher ops burden, used only if vendor costs/availability become blockers, or as a second-opinion cross-check.

**Recommendation:** start with a specialist African-ID vendor if one supports Zimbabwe National ID out of the box (fastest time-to-accuracy); otherwise AWS Textract AnalyzeID + our own regex/checksum validators layered on top to compensate for template gaps.

### 5.3 Processing flow

1. Orchestrator receives `media_id`s from the request (never raw files).
2. Calls Media Service (or uses the signed URL already provided) to get a short-lived URL for the OCR provider to fetch — OCR provider pulls the image directly rather than us proxying bytes where possible, to avoid handling raw image bytes in our service.
3. OCR provider returns structured fields + confidence.
4. Our validators run the Zimbabwe-specific checks (checksum, expiry, regex format).
5. Aggregate confidence = weighted average of OCR confidence and validator pass/fail.
6. Decision:
   - confidence ≥ 0.90 and all validators pass → `approved`
   - confidence between 0.60–0.90 or a soft validator warning → `needs_review` (manual queue)
   - confidence < 0.60 or hard validator failure (checksum fails, expired doc) → `rejected`

---

## 6. API Design

### 6.1 REST endpoints (v1)

```
POST   /v1/verifications
       body: { user_id, document_type, front_media_id, back_media_id? }
       → 202 Accepted, { verification_id, status: "pending" }

GET    /v1/verifications/{id}
       → { verification_id, status, confidence, rejection_reason?, extracted_fields? }

GET    /v1/users/{user_id}/verifications/latest
       → most recent verification + status

POST   /v1/verifications/{id}/selfie   (phase 2, face match)
       body: { selfie_media_id }

POST   /v1/verifications/{id}/review   (internal, manual review tooling)
       body: { decision: "approve"|"reject", reviewer_id, notes }
```

### 6.2 Async processing

`POST /v1/verifications` should return immediately with `pending` and do OCR asynchronously (job queued to a worker pool or a message queue), since OCR provider round-trips can take 1–5s and we don't want to hold client connections open. Client polls `GET` or subscribes to a webhook/event.

### 6.3 Events published (Kafka/NATS)

```
verification.completed   { verification_id, user_id, status, document_type }
verification.needs_review { verification_id, user_id }
```

Downstream services (User Service, Store Service) subscribe to `verification.completed` to unlock gated features once `status == approved`.

---

## 7. Security & Compliance

- **Legal basis:** Zimbabwe's Cyber and Data Protection Act (2021) governs processing of personal data, including ID numbers, and requires a lawful basis, data-subject notice, and breach reporting. Confirm with legal counsel whether formal registration as a data controller with the relevant regulator is required before launch.
- **PII minimization:** we store extracted fields and document numbers, never the raw image bytes (those live only in Media Service). Even so, document numbers and DOB are still sensitive — encrypt at rest (pgcrypto or application-level AES-GCM) and restrict column access.
- **Uniqueness without plaintext exposure everywhere:** use an HMAC-SHA256 (server-side secret key) of the normalized document number for the uniqueness index; keep the encrypted plaintext separately for cases where a human reviewer needs to see it.
- **Access control:** verification records and extracted fields are only queryable by the owning user and by an internal reviewer role — enforce via RBAC middleware, not just at the DB layer.
- **Audit log:** every status transition and every manual review action is appended to an immutable audit table (`verification_audit_log`) with actor, timestamp, before/after status.
- **Data retention:** define a retention period (e.g. retain extracted fields for the life of the account + N years per legal requirement; media itself governed by Media Service's own retention policy) and a deletion job.
- **Transport:** mTLS or signed URLs only between this service, Media Service, and the OCR provider — no long-lived public URLs to ID documents.

---

## 8. Error Handling & Manual Review

- OCR provider timeout/5xx → retry with exponential backoff (max 3 attempts), then mark `needs_review` rather than `rejected` — never auto-reject on infra failure.
- Low-confidence or ambiguous validator results go to a **manual review queue** (internal tool, simple list view + approve/reject with reason codes) rather than blocking the user indefinitely.
- User-facing rejection reasons should be actionable but not overly specific about internal thresholds, e.g. "We couldn't clearly read your ID. Please retake the photo in better lighting" rather than "confidence 0.42."

---

## 9. Anti-Abuse / Rate Limiting

- Rate-limit verification attempts per user (e.g. max 5 attempts/day) to prevent brute-forcing document uniqueness checks or spamming the OCR provider (cost control).
- Flag (not necessarily block) accounts that submit the same document number as an existing verified account under a different user_id — route to manual review, since this may indicate account takeover or a legitimate re-registration.
- Image quality pre-check (blur/glare/crop detection) before sending to the OCR provider, to save cost on obviously bad submissions — many OCR vendors expose this as part of the same call.

---

## 10. Tech Stack

| Concern | Choice |
|---|---|
| Language | Go 1.22+ |
| API | REST via `chi` or `net/http`; gRPC internal if other services need low-latency calls |
| DB | PostgreSQL (pgx v5) |
| Queue/Events | Kafka or NATS JetStream (align with existing platform choice) |
| OCR | Vendor SDK/HTTP client behind `OCRProvider` interface |
| Object storage | None directly — delegated to Media Service |
| Observability | OpenTelemetry traces, Prometheus metrics, structured logging (zerolog/slog) |
| Migrations | `golang-migrate` or `atlas` |
| Architecture pattern | Clean architecture: `handler → usecase (orchestrator) → repository/adapters`, matching your existing Go services |

---

## 11. Rollout Plan

**Phase 1 — Core OCR verification (MVP)**
- National ID + Passport support, one OCR vendor, synchronous validators, manual review queue.

**Phase 2 — Driver's Licence + Face Match**
- Add driver's licence validators; add selfie capture + face-match adapter (liveness check) for higher-trust tiers (e.g. sellers/payouts).

**Phase 3 — Optimization**
- Add a second OCR vendor for A/B accuracy comparison / fallback; introduce re-verification expiry cadence; explore self-hosted OCR to reduce per-call cost at scale.

**Phase 4 — Registry integration (stretch)**
- Revisit direct integration with the Registrar-General or a licensed local KYC bureau if/when an API becomes available, to move from "document looks valid" to "document is confirmed against source-of-truth."

---

## 12. Open Questions / Risks

- Which OCR vendor actually has good accuracy on the Zimbabwean National ID layout? Needs a small accuracy bake-off with real (or representative synthetic) sample images before committing.
- Do we need explicit consent capture/versioning per the Cyber and Data Protection Act before running biometric face-match (this is typically treated as sensitive personal data)?
- What's the SLA/cost ceiling per verification call from the chosen OCR vendor, and does it scale with expected verification volume?
- Manual review team sizing/tooling — who reviews `needs_review` cases and what's the target turnaround time (this directly affects user-facing "verification pending" duration)?
