-- Create the new enum type
CREATE TYPE verification_status AS ENUM ('pending', 'processing', 'rejected', 'verified');

-- Add the new column with the enum type, defaulting to 'pending'
ALTER TABLE users
    ADD COLUMN verification_status verification_status NOT NULL DEFAULT 'pending';

-- Migrate existing data: is_verified = TRUE → 'verified', FALSE → 'pending'
UPDATE users SET verification_status = 'verified' WHERE is_verified = TRUE;
UPDATE users SET verification_status = 'pending'  WHERE is_verified = FALSE;

-- Drop the old boolean column
ALTER TABLE users DROP COLUMN is_verified;

-- Rename new column to is_verified for minimal downstream impact
ALTER TABLE users RENAME COLUMN verification_status TO is_verified;
