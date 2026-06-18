-- Rename column back to verification_status temporarily
ALTER TABLE users RENAME COLUMN is_verified TO verification_status;

-- Add back the boolean column
ALTER TABLE users ADD COLUMN is_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- Migrate back: 'verified' → TRUE, everything else → FALSE
UPDATE users SET is_verified = TRUE  WHERE verification_status = 'verified';
UPDATE users SET is_verified = FALSE WHERE verification_status != 'verified';

-- Drop the enum column
ALTER TABLE users DROP COLUMN verification_status;

-- Drop the enum type
DROP TYPE verification_status;
