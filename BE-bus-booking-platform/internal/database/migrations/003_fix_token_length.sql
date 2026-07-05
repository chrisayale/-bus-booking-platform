-- Increase token column sizes for QR and download tokens
ALTER TABLE tickets ALTER COLUMN qr_token TYPE TEXT;
ALTER TABLE tickets ALTER COLUMN download_token TYPE TEXT;
ALTER TABLE users ALTER COLUMN verification_token TYPE TEXT;
ALTER TABLE users ALTER COLUMN reset_token TYPE TEXT;
