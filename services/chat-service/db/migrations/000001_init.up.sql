CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL DEFAULT 'THREAD_TYPE_DIRECT',
    title VARCHAR(255),
    buyer_id VARCHAR(255),
    seller_id VARCHAR(255) NOT NULL,
    order_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for querying threads by buyer or seller
CREATE INDEX idx_threads_buyer_id ON threads (buyer_id);
CREATE INDEX idx_threads_seller_id ON threads (seller_id);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    sender_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'MESSAGE_TYPE_TEXT',
    content TEXT NOT NULL,
    media_url VARCHAR(1024),
    reference_id VARCHAR(255),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_messages_thread_id ON messages (thread_id);
