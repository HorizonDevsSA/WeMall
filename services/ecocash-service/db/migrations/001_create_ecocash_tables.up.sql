CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Payment Transactions ──────────────────────────────────────────────────────
-- Stores every MER charge attempt. client_correlator is the idempotency key;
-- it is generated before calling EcoCash and reused on retries.
CREATE TABLE payment_transactions (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id                UUID        NOT NULL,
    client_correlator       VARCHAR(64) NOT NULL UNIQUE,
    reference_code          VARCHAR(64) NOT NULL DEFAULT '',
    tran_type               VARCHAR(8)  NOT NULL DEFAULT 'MER',
    -- end_user_id (MSISDN) stored as pgcrypto-encrypted hex so PII never sits in plaintext.
    -- Use pgp_sym_encrypt(msisdn, current_setting('app.encryption_key')) to write,
    -- pgp_sym_decrypt(end_user_id::bytea, ...) to read. For MVP we store masked form.
    end_user_id_masked      VARCHAR(20) NOT NULL, -- e.g. 773***653
    amount_cents            BIGINT      NOT NULL,
    currency                VARCHAR(3)  NOT NULL,
    status                  VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    ecocash_status_code     VARCHAR(8),
    ecocash_status_msg      TEXT,
    ecocash_transaction_id  VARCHAR(64),
    raw_request             JSONB,
    raw_response            JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payment_transactions_tran_type_check CHECK (tran_type IN ('MER', 'REF', 'REV')),
    CONSTRAINT payment_transactions_status_check    CHECK (status IN (
        'PENDING', 'SUCCESS', 'FAILED', 'REFUNDED', 'REVERSED', 'MANUAL_REVIEW'
    ))
);

CREATE INDEX idx_ptxn_order_id    ON payment_transactions(order_id);
CREATE INDEX idx_ptxn_status      ON payment_transactions(status);
CREATE INDEX idx_ptxn_created_at  ON payment_transactions(created_at);

-- ── Refund / Reversal Transactions ────────────────────────────────────────────
CREATE TABLE refund_transactions (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    original_txn_id         UUID        NOT NULL REFERENCES payment_transactions(id),
    client_correlator       VARCHAR(64) NOT NULL UNIQUE,
    tran_type               VARCHAR(8)  NOT NULL DEFAULT 'REF', -- REF or REV
    amount_cents            BIGINT      NOT NULL,
    status                  VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    ecocash_status_code     VARCHAR(8),
    ecocash_status_msg      TEXT,
    raw_request             JSONB,
    raw_response            JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT refund_transactions_tran_type_check CHECK (tran_type IN ('REF', 'REV')),
    CONSTRAINT refund_transactions_status_check    CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED'))
);

CREATE INDEX idx_rtxn_original_txn ON refund_transactions(original_txn_id);

-- ── Payouts ───────────────────────────────────────────────────────────────────
-- Manual settlement bridge: finance team executes bulk transfers via EcoCash
-- merchant portal/bank rail; this table tracks the request and reconciliation.
CREATE TABLE payouts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id    UUID        NOT NULL,
    amount_cents BIGINT      NOT NULL,
    currency     VARCHAR(3)  NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'QUEUED',
    method       VARCHAR(20) NOT NULL DEFAULT 'manual_bulk',
    provider_ref VARCHAR(64),
    raw_response JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payouts_status_check CHECK (status IN (
        'QUEUED', 'PROCESSING', 'PAID', 'FAILED', 'MANUAL_REVIEW'
    ))
);

CREATE INDEX idx_payouts_seller_id ON payouts(seller_id);
CREATE INDEX idx_payouts_status    ON payouts(status);

-- ── Transactional Outbox ──────────────────────────────────────────────────────
-- DB row + outbox written in same transaction; relay worker publishes to NATS
-- and marks published=true. Guarantees no lost events if NATS is briefly down.
CREATE TABLE outbox_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID        NOT NULL,
    event_type   VARCHAR(64) NOT NULL,
    payload      JSONB       NOT NULL,
    published    BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_unpublished ON outbox_events(published, created_at) WHERE published = false;
