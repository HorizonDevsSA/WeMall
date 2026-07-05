-- Migration to support enterprise chat features

-- 1. Alter threads table to make seller_id optional and add delivery/courier/support participants
ALTER TABLE threads ALTER COLUMN seller_id DROP NOT NULL;
ALTER TABLE threads ADD COLUMN delivery_boy_id VARCHAR(255);
ALTER TABLE threads ADD COLUMN courier_station_id VARCHAR(255);
ALTER TABLE threads ADD COLUMN support_agent_id VARCHAR(255);

-- 2. Create indices for faster participant lookups
CREATE INDEX idx_threads_delivery_boy_id ON threads (delivery_boy_id);
CREATE INDEX idx_threads_courier_station_id ON threads (courier_station_id);
CREATE INDEX idx_threads_support_agent_id ON threads (support_agent_id);

-- 3. Create thread_members table to support group/broadcast/system announcement memberships
CREATE TABLE thread_members (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'MEMBER',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (thread_id, user_id)
);

CREATE INDEX idx_thread_members_user_id ON thread_members (user_id);

-- 4. Add metadata JSONB column to messages table for products, coupons, and promotions
ALTER TABLE messages ADD COLUMN metadata JSONB;
