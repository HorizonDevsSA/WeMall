CREATE TYPE media_status AS ENUM (
    'pending_upload',  -- Presigned S3 URL generated, waiting for client upload
    'uploaded',        -- Client confirmed S3 upload, ready for processing
    'processing',      -- Lambda or MediaConvert is converting resources
    'completed',       -- Asset variants successfully written to destination S3
    'failed'           -- Error occurred during validation, transcoding, or storage
);

CREATE TABLE media_assets (
    id             UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id       UUID             NOT NULL,               -- User/Seller ID who uploaded
    service_scope  VARCHAR(50)      NOT NULL,               -- 'user-avatar' | 'product-image' | 'seller-kyc'
    original_name  VARCHAR(255)     NOT NULL,               -- e.g. "profile_pic.jpg"
    mime_type      VARCHAR(100)     NOT NULL,               -- e.g. "image/jpeg"
    size_bytes     BIGINT           NOT NULL,               -- Size of raw upload in bytes
    raw_s3_key     VARCHAR(512)     NOT NULL,               -- Key in wemall-media-raw
    is_private     BOOLEAN          NOT NULL DEFAULT FALSE, -- Determines S3 destination & CF signed access
    status         media_status     NOT NULL DEFAULT 'pending_upload',
    variants       JSONB,                                   -- Stores JSON of URL optimized maps
    error_message  TEXT,                                    -- Populated if status is 'failed'
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_media_assets_owner ON media_assets(owner_id);
CREATE INDEX idx_media_assets_status ON media_assets(status) WHERE status != 'completed';
