# Real-Name Verification Service — Zimbabwe Market
### Technical Design & Implementation Plan (Go) — v2, Production-Ready

**Owner:** Backend Engineering
**Status:** Draft v2 (supersedes v1)
**Scope:** KYC / identity verification microservice for Passport, National ID, and Driver's Licence, targeting Zimbabwean users.

---

## Changelog from v1 — Loopholes Fixed

| # | Issue in v1 | Fix in v2 |
|---|---|---|
| 1 | Auto-`approved` on OCR confidence alone — no proof the submitter *is* the document holder | Trust-tiered rollout: document-only unlocks low-risk actions; payouts/selling require face-match + liveness before Phase 1 exits for those features |
| 2 | `user_id` accepted from request body | `user_id` derived only from authenticated session/JWT |
| 3 | Unique index on plaintext `document_number`; contradicts "flag not block" | HMAC-based composite unique key; conflict caught in usecase layer → routed to `needs_review`, never a raw DB error |
| 4 | No data-residency stance for OCR vendor | Explicit residency/DPA check gates vendor selection |
| 5 | No submission idempotency | Idempotency key + partial unique index on in-flight verifications |
| 6 | Dual-write risk between Postgres and Kafka | Transactional outbox pattern |
| 7 | Signed media URL may expire before async OCR call | Worker fetches a fresh URL at call time |
| 8 | `review` API has `notes`, schema doesn't | Schema updated to match contract |
| 9 | No age-of-majority check | Added as a hard validator |
| 10 | Extracted PII returned to client in `GET` | Client response redacted to status/tier only |
| 11 | `ExpiresAt` field with no process behind it | Scheduled re-verification/expiry worker specified |

---

## 1. Goals & Non-Goals

### Goals
- Verify that a user is a real, uniquely-identifiable person **and that the submitter is that person** before granting privileged access (selling, high-value transactions, payouts).
- Support National ID, Passport (MRZ), and Driver's Licence.
- Extract structured data from document images via OCR.
- Perform liveness/selfie-to-document face match — **mandatory gate for payout/seller tiers, not an optional phase-2 nicety.**
- Produce a durable, auditable verification record with status and confidence score, with every transition attributable to a cause (system decision or named reviewer).
- Be provider-agnostic for OCR/face-match.

### Non-Goals (v1 scope)
- Building in-house OCR/ML models.
- Storing raw document images ourselves (delegated to Media Service; we store URL + metadata + hashes only).
- Real-time integration with Zimbabwe's national ID registry (flagged as Phase 4 risk).

---

## 2. High-Level Architecture

```
                     ┌───────────────────────┐
                     │      Client App        │
                     └──────────┬─────────────┘
                                │ 1. Upload doc images (auth'd session)
                                ▼
                     ┌───────────────────────┐
                     │     Media Service       │  (existing, out of scope)
                     │  stores file, returns   │
                     │  signed URL + media_id  │
                     └──────────┬─────────────┘
                                │ 2. media_id (+ URL, but re-fetched at OCR time)
                                ▼
          ┌────────────────────────────────────────────┐
          │        Identity Verification Service         │  ← this plan
          │                (Go, modular monolith)         │
          │                                              │
          │  ┌────────────┐  ┌────────────┐  ┌─────────┐ │
          │  │  API Layer  │→│ Verification │→│  OCR     │ │
          │  │ (REST/gRPC) │  │ Orchestrator│  │ Provider │ │
          │  │ user_id from│  │ (idempotent)│  │ Adapter  │ │
          │  │ auth ctx    │  └─────┬───────┘  └─────────┘ │
          │  └────────────┘        │                       │
          │                        ▼                       │
          │                ┌───────────────┐               │
          │                │ Rules Engine /  │              │
          │                │ Validators      │              │
          │                │ (per doc type + │              │
          │                │  age, expiry)   │              │
          │                └───────┬─────────┘              │
          │                        ▼                        │
          │                ┌───────────────┐                │
          │                │  Face Match     │  mandatory     │
          │                │  + Liveness     │  for payout/   │
          │                │  Adapter        │  seller tier   │
          │                └───────┬─────────┘               │
          │                        ▼                          │
          │                ┌───────────────┐                  │
          │                │ Postgres:       │                 │
          │                │ verifications,  │                 │
          │                │ extracted_fields,│                │
          │                │ outbox_events   │                 │
          │                └───────┬─────────┘                 │
          │                        ▼                            │
          │                ┌───────────────┐                    │
          │                │ Outbox Relay →  │  → Kafka/NATS      │
          │                │ Event Bus       │    notifies other  │
          │                └───────────────┘    services         │
          └────────────────────────────────────────────┘
                                │ 3. status event (verification_id, status only — no PII)
                                ▼
                     ┌───────────────────────┐
                     │  Downstream Services    │
                     │ (User/Store/Payments)   │
                     └───────────────────────┘
```

**Deployment shape:** single Go service to start (modular monolith, clean architecture layers), split into `ocr-worker` once volume justifies it.

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
    StatusApproved         VerificationStatus = "approved"          // document-only tier
    StatusApprovedFullTier VerificationStatus = "approved_full"     // document + face-match tier
    StatusRejected         VerificationStatus = "rejected"
    StatusExpired          VerificationStatus = "expired"
)

type TrustTier string

const (
    TierDocumentOnly TrustTier = "document_only" // browsing, low-value actions
    TierFullVerified TrustTier = "full_verified" // selling, payouts, high-value tx
)

type Verification struct {
    ID              uuid.UUID
    UserID          string          // ALWAYS from auth context, never client input
    IdempotencyKey  string          // client-supplied or derived; unique per user+doc type in-flight
    DocumentType    DocumentType
    FrontMediaID    string
    BackMediaID     *string
    SelfieMediaID   *string
    Status          VerificationStatus
    TrustTier       TrustTier
    Confidence      float64
    RejectionReason *string
    ReviewedBy      *string
    ReviewNotes     *string
    SupersedesID    *uuid.UUID      // links resubmission to the prior attempt
    CreatedAt       time.Time
    UpdatedAt       time.Time
    ReviewedAt      *time.Time
    ExpiresAt       *time.Time
}

// ExtractedFields — same shape as v1, plus document number stored as
// hash + encrypted ciphertext, never plaintext in a queryable column.
type ExtractedFields struct {
    VerificationID       uuid.UUID
    FullName             string
    DateOfBirth          *time.Time
    Sex                  *string
    DocumentNumberHMAC   string          // deterministic, for uniqueness lookups
    DocumentNumberEnc    []byte          // AES-GCM ciphertext, decrypted only for review UI
    Nationality          *string
    IssueDate            *time.Time
    ExpiryDate           *time.Time
    IssuingAuthority     *string
    RawOCRPayload        json.RawMessage
    OCRConfidence        float64
    ValidatorVersion     string          // which ruleset produced the decision, for audit
}
```

### 3.2 Postgres Tables

```sql
CREATE TABLE verifications (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            TEXT NOT NULL,
    idempotency_key    TEXT NOT NULL,
    document_type      TEXT NOT NULL,
    front_media_id     TEXT NOT NULL,
    back_media_id      TEXT,
    selfie_media_id    TEXT,
    status             TEXT NOT NULL DEFAULT 'pending',
    trust_tier         TEXT NOT NULL DEFAULT 'document_only',
    confidence         NUMERIC(4,3),
    rejection_reason   TEXT,
    reviewed_by        TEXT,
    review_notes       TEXT,
    supersedes_id      UUID REFERENCES verifications(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at        TIMESTAMPTZ,
    expires_at         TIMESTAMPTZ
);

-- Prevents duplicate in-flight submissions for the same user+doc type
CREATE UNIQUE INDEX idx_verifications_in_flight
    ON verifications (user_id, document_type)
    WHERE status IN ('pending', 'processing');

CREATE UNIQUE INDEX idx_verifications_idempotency
    ON verifications (user_id, idempotency_key);

CREATE INDEX idx_verifications_user_id ON verifications(user_id);
CREATE INDEX idx_verifications_status ON verifications(status);

CREATE TABLE extracted_fields (
    verification_id      UUID PRIMARY KEY REFERENCES verifications(id),
    full_name             TEXT,
    date_of_birth          DATE,
    sex                   TEXT,
    document_number_hmac  TEXT NOT NULL,   -- HMAC-SHA256(secret, normalized_number)
    document_number_enc   BYTEA NOT NULL,  -- AES-GCM ciphertext
    nationality           TEXT,
    issue_date            DATE,
    expiry_date           DATE,
    issuing_authority     TEXT,
    raw_ocr_payload       JSONB,
    ocr_confidence        NUMERIC(4,3),
    validator_version     TEXT NOT NULL
);

-- Uniqueness enforced on the HASH, scoped per document type, never on plaintext.
-- Handle the conflict in application code (see §6.2) and route to needs_review
-- instead of letting the insert fail the request outright.
CREATE UNIQUE INDEX idx_document_hash_type
    ON extracted_fields (document_type, document_number_hmac);

-- Transactional outbox: written in the SAME transaction as any verification
-- status change; a relay process publishes and marks dispatched.
CREATE TABLE outbox_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id  UUID NOT NULL,          -- verification_id
    event_type    TEXT NOT NULL,          -- e.g. 'verification.completed'
    payload       JSONB NOT NULL,         -- status-only, no PII
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at TIMESTAMPTZ
);
CREATE INDEX idx_outbox_undispatched ON outbox_events (created_at)
    WHERE dispatched_at IS NULL;

CREATE TABLE verification_audit_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    verification_id  UUID NOT NULL REFERENCES verifications(id),
    actor            TEXT NOT NULL,   -- 'system' or reviewer_id, from auth context
    action           TEXT NOT NULL,
    before_status     TEXT,
    after_status      TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

> `document_number_hmac` uses a server-side secret held in KMS (see §7). The plaintext is only ever decrypted server-side for a reviewer viewing a `needs_review` case, and that decrypt action is itself audit-logged.

---

## 4. Zimbabwe Document Specifics

| Document | Key fields to extract | Format notes | Validation |
|---|---|---|---|
| **National ID** | ID number, full name, DOB, sex, district | `NN-NNNNNNN-X-NN` | Regex + check-letter algorithm + **age ≥ 18 validator** |
| **Passport** | Full name, passport no., DOB, sex, nationality, expiry, MRZ | ICAO 9303 2-line MRZ | MRZ composite checksum + **age ≥ 18 validator** |
| **Drivers Licence** | Licence no., full name, DOB, classes, expiry | VID card, front/back | Expiry in future; class codes validated; **age ≥ 18 validator** |

```go
type Validator interface {
    Validate(fields ExtractedFields) (ValidationResult, error)
}

type ValidationResult struct {
    Valid       bool
    Confidence  float64
    Issues      []string // "checksum_failed", "expired", "underage", "dob_mismatch"
}
```

Every validator run records `ValidatorVersion` against the result, so a later dispute ("why was this rejected?") can be traced to the exact ruleset in force at decision time — not just today's rules re-applied retroactively.

National ID checksum and passport MRZ checksum remain pure functions we implement ourselves — no external dependency, and the single most valuable Zimbabwe-specific piece of logic here.

---

## 5. OCR & Face-Match Strategy

### 5.1 Provider abstraction

```go
type OCRProvider interface {
    ExtractDocument(ctx context.Context, req OCRRequest) (OCRResult, error)
}

type FaceMatchProvider interface {
    MatchAndCheckLiveness(ctx context.Context, req FaceMatchRequest) (FaceMatchResult, error)
}

type OCRRequest struct {
    DocumentType  DocumentType
    FrontImageURL string  // fetched fresh at call time, not the ingestion-time URL
    BackImageURL  *string
}
```

### 5.2 Candidate providers — data residency now a first-class filter

1. **Regional African KYC specialists** (Smile Identity, Youverify) — some have explicit Zimbabwe National ID support, combine OCR + liveness + face-match in one call, **and are more likely to have a data-processing posture compatible with the Cyber and Data Protection Act.** Evaluate first, not third.
2. **AWS Textract (AnalyzeID)** / **Google Document AI** — strong general accuracy, but confirm processing region and get a signed DPA before committing; cross-border transfer of Zimbabwean national ID data to a US/EU processor needs a documented legal basis, not an assumption.
3. **Self-hosted fallback** (Tesseract/PaddleOCR) — higher ops burden, kept as a cost/availability fallback or cross-check.

**Recommendation:** resolve the data-residency question *before* the accuracy bake-off, since it may eliminate options 1–2 above regardless of accuracy.

### 5.3 Processing flow

1. Orchestrator receives `media_id`s (never raw files), with `user_id` taken from the authenticated request context — a client cannot submit on another user's behalf.
2. Idempotency check: if an in-flight verification exists for `(user_id, document_type)`, return the existing `verification_id` rather than creating a duplicate.
3. Worker fetches a **fresh** short-lived signed URL from Media Service immediately before calling the OCR provider (not the URL captured at ingestion, which may have expired by the time the job runs).
4. OCR provider returns structured fields + confidence.
5. Zimbabwe-specific validators run (checksum, expiry, regex, **age**).
6. Document-number uniqueness check against `document_number_hmac`: on conflict, do **not** fail the request — catch it in the usecase layer, set status to `needs_review` with reason `duplicate_document`, and continue (per §9).
7. Decision for **document-only tier**:
   - confidence ≥ 0.90 and all validators pass → `approved` (`document_only` tier)
   - confidence 0.60–0.90, or a soft validator warning, or a uniqueness conflict → `needs_review`
   - confidence < 0.60 or hard validator failure (checksum fails, expired, underage) → `rejected`
8. **Full-verified tier is never granted on OCR confidence alone.** It requires an additional passing `FaceMatchProvider.MatchAndCheckLiveness` call comparing the selfie to the document photo, with its own liveness-spoof check (not just similarity score). A user attempting to sell or receive payouts is routed to selfie capture before that tier unlocks, regardless of how confident the document OCR was.

---

## 6. API Design

### 6.1 REST endpoints (v1)

```
POST   /v1/verifications
       auth: required (user_id derived from session, never accepted in body)
       header: Idempotency-Key (optional; derived from user_id+document_type if absent)
       body: { document_type, front_media_id, back_media_id? }
       → 202 Accepted, { verification_id, status: "pending" }

GET    /v1/verifications/{id}
       auth: required (owner or internal reviewer role only)
       → { verification_id, status, trust_tier, confidence_tier: "low"|"medium"|"high",
           rejection_reason? }
       # Note: extracted PII (DOB, document number, full name) is NEVER returned
       # to the end-user client. It is only visible via the internal review API.

GET    /v1/users/{user_id}/verifications/latest
       auth: required (owner or internal reviewer role only)

POST   /v1/verifications/{id}/selfie
       body: { selfie_media_id }
       → triggers face-match/liveness check; on pass, tier upgrades to full_verified

POST   /v1/verifications/{id}/review   (internal only, RBAC-gated)
       body: { decision: "approve"|"reject", notes }
       # reviewer_id taken from the authenticated internal session, never from body

GET    /v1/internal/verifications/{id}/pii   (internal reviewer only, itself audit-logged)
       → decrypted extracted_fields, for manual review UI use only
```

### 6.2 Async processing & idempotency

`POST /v1/verifications` returns `pending` immediately; OCR/validation run async via worker pool. On retry with the same idempotency key (or same `user_id`+`document_type` while a verification is still in-flight), the existing `verification_id` is returned rather than creating a duplicate job — this also protects against double-billing the OCR vendor on client retries.

When the OCR/validation step hits the `document_number_hmac` uniqueness conflict from §3.2, the usecase layer catches the constraint violation explicitly, sets `status = needs_review`, `rejection_reason = "duplicate_document"`, and proceeds — it does not propagate a raw DB error back through the API.

### 6.3 Events published (Kafka/NATS, via outbox relay)

```
verification.completed    { verification_id, user_id, status, trust_tier, document_type }
verification.needs_review { verification_id, user_id, reason }
```

Payloads are intentionally status-only — no document numbers, no DOB, no names — since these events fan out to multiple downstream services and widen the PII blast radius otherwise.

---

## 7. Security & Compliance

- **Legal basis:** Cyber and Data Protection Act (2021) governs this data. Confirm data-controller registration requirement with legal counsel before launch.
- **Cross-border transfer:** if the chosen OCR/face-match vendor processes data outside Zimbabwe, get an explicit legal sign-off (DPA, SCCs, or equivalent) before sending any production document images — this gates vendor selection, not just a post-launch risk note.
- **PII minimization:** raw images live only in Media Service. `document_number` is stored as HMAC (for uniqueness) + AES-GCM ciphertext (for review), never plaintext, never in a queryable column.
- **Key management:** HMAC secret and AES keys live in KMS, not application config; define a rotation policy and who (which service account/role) is authorized to decrypt.
- **Access control:** enforced by RBAC middleware, not just DB grants. `user_id` is always derived from the authenticated session — never trusted from a request body — closing the account-spoofing gap in v1.
- **Audit log:** every status transition, every manual review action, and every PII decrypt is appended to `verification_audit_log` with actor, timestamp, before/after state.
- **Data retention & erasure:** retain extracted fields per legal requirement; support user-initiated deletion requests, while retaining the HMAC (not the plaintext) for fraud-prevention uniqueness checks even after account deletion, since the hash alone doesn't reconstitute PII.
- **Transport:** mTLS or signed URLs only between this service, Media Service, and OCR/face-match providers — no long-lived public URLs to ID documents.

---

## 8. Error Handling & Manual Review

- OCR/face-match provider timeout/5xx → retry with exponential backoff (max 3 attempts, idempotent to the vendor call), then `needs_review` — never auto-reject on infra failure.
- Uniqueness conflicts, low confidence, and soft validator warnings all route to the manual review queue rather than blocking the user indefinitely.
- User-facing rejection reasons stay actionable but non-specific about thresholds ("We couldn't clearly read your ID — please retake the photo in better lighting"), never surfacing confidence scores or internal reason codes.
- Reviewer decisions require `notes` (schema now supports this — v1 had a mismatch here) and are captured with `reviewed_at`/`reviewed_by` from the authenticated internal session.

---

## 9. Anti-Abuse / Rate Limiting

- Rate-limit verification attempts per user (e.g. max 5/day) to control OCR vendor cost and blunt brute-force uniqueness probing.
- Duplicate document number (same HMAC) under a different `user_id` → **flagged to manual review, and the insert conflict itself is handled gracefully in the usecase layer** (§6.2) rather than surfacing as a hard DB failure to the requesting user. This closes the v1 contradiction between "flag not block" and an unhandled unique-constraint violation.
- Image quality pre-check (blur/glare/crop) before sending to the OCR provider, to save cost on obviously bad submissions.
- Basic bot/automation signal (e.g. request velocity, device fingerprint if already collected elsewhere in the platform) feeding into the same review queue, not a hard block, to avoid false positives against legitimate low-connectivity users retrying uploads.

---

## 10. Tech Stack

| Concern | Choice |
|---|---|
| Language | Go 1.22+ |
| API | REST via `chi` or `net/http`; gRPC internal if needed |
| DB | PostgreSQL (pgx v5) |
| Queue/Events | Kafka or NATS JetStream, via transactional outbox |
| OCR / Face-Match | Vendor SDK/HTTP client behind `OCRProvider` / `FaceMatchProvider` interfaces |
| Secrets/Keys | KMS-backed, rotated; never in application config |
| Object storage | None directly — delegated to Media Service |
| Observability | OpenTelemetry traces, Prometheus metrics, structured logging (zerolog/slog), alerting on review-queue backlog and rejection-rate spikes |
| Migrations | `golang-migrate` or `atlas` |
| Scheduler | Cron-driven worker for `ExpiresAt` sweep → `expired` transition + re-verification event |
| Architecture pattern | Clean architecture: `handler → usecase (orchestrator) → repository/adapters` |

---

## 11. Rollout Plan

**Phase 1 — Core document verification (document-only tier)**
- National ID + Passport, one OCR vendor (residency-cleared), validators including age check, idempotent submission, manual review queue, transactional outbox.

**Phase 1.5 — Face-match gate for full tier (do not defer past MVP for payout-gated features)**
- Selfie capture + liveness/face-match required before any user reaches `full_verified` trust tier. This ships alongside Phase 1 for any feature that touches money, even if it lags for browsing-only features.

**Phase 2 — Driver's Licence + expiry automation**
- Add licence validators; scheduled worker for `ExpiresAt` sweep and re-verification prompts.

**Phase 3 — Optimization**
- Second OCR vendor for A/B accuracy/fallback; explore self-hosted OCR at scale.

**Phase 4 — Registry integration (stretch)**
- Direct integration with the Registrar-General or a licensed local KYC bureau, if/when available.

---

## 12. Open Questions / Risks

- Which OCR vendor has good Zimbabwean National ID accuracy **and** an acceptable data-residency/DPA posture — both conditions, not accuracy alone.
- Explicit consent capture/versioning for biometric face-match, likely treated as sensitive personal data under the Act.
- SLA/cost ceiling per verification+face-match call, and how it scales with volume.
- Manual review team sizing and target turnaround — this directly drives user-facing "pending" duration and, combined with the flag-not-block duplicate handling, review queue volume.
- Right-to-erasure requests vs. retained HMAC for fraud prevention — confirm with legal counsel that hash-only retention post-deletion satisfies the Act's requirements.
