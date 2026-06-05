CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('buyer', 'seller', 'admin');
CREATE TYPE auth_provider AS ENUM ('email', 'google', 'phone');

CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT         UNIQUE,
    phone         TEXT         UNIQUE,
    password_hash TEXT,
    full_name     TEXT         NOT NULL,
    avatar_url    TEXT,
    role          user_role    NOT NULL DEFAULT 'buyer',
    auth_provider auth_provider NOT NULL DEFAULT 'email',
    is_verified   BOOLEAN      NOT NULL DEFAULT FALSE,
    google_id     TEXT         UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT email_or_phone CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

CREATE INDEX idx_users_email     ON users(email)     WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone     ON users(phone)     WHERE deleted_at IS NULL;
CREATE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
