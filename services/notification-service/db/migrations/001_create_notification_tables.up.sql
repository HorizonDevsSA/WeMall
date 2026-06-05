-- Create Enum for Notification Categories
CREATE TYPE notification_category AS ENUM (
    'transactional',
    'security',
    'low_stock',
    'follows',
    'marketing'
);

-- Create Enum for Delivery Status
CREATE TYPE delivery_status AS ENUM (
    'queued',
    'sent',
    'failed',
    'retrying'
);

-- 1. Device Tokens (for Firebase FCM Push Notifications)
CREATE TABLE user_device_tokens (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    token       TEXT         NOT NULL UNIQUE,
    platform    TEXT         NOT NULL,
    device_name TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_device_tokens_user_id ON user_device_tokens(user_id);

-- 2. User Notification Preferences (Opt-in / Opt-out Settings)
CREATE TABLE user_notification_preferences (
    id                  UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID                  NOT NULL,
    category            notification_category NOT NULL,
    email_enabled       BOOLEAN               NOT NULL DEFAULT TRUE,
    push_enabled        BOOLEAN               NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category)
);
CREATE INDEX idx_preferences_user_id ON user_notification_preferences(user_id);

-- 3. Push Notifications (Stores sent FCM payloads for user in-app notification center)
CREATE TABLE push_notifications (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL,
    token       TEXT         NOT NULL,
    title       TEXT         NOT NULL,
    body        TEXT         NOT NULL,
    payload     JSONB        NOT NULL,
    is_read     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    read_at     TIMESTAMPTZ
);
CREATE INDEX idx_push_notifications_user_unread ON push_notifications(user_id) WHERE is_read = FALSE;
CREATE INDEX idx_push_notifications_created_at ON push_notifications(created_at DESC);

-- 4. Notification Log (Audit Trail & Outbox pattern for all channels)
CREATE TABLE notification_logs (
    id             UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID             NOT NULL,
    category       TEXT             NOT NULL,
    channel        TEXT             NOT NULL,          -- 'email' | 'push'
    recipient      TEXT             NOT NULL,          -- email address or device token
    title          TEXT             NOT NULL,
    content        TEXT             NOT NULL,
    status         delivery_status  NOT NULL DEFAULT 'queued',
    retry_count    INTEGER          NOT NULL DEFAULT 0,
    error_message  TEXT,
    sent_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notif_logs_user_status ON notification_logs(user_id, status);
CREATE INDEX idx_notif_logs_created_at ON notification_logs(created_at);
