-- Bus images stored as BYTEA in PostgreSQL
CREATE TABLE IF NOT EXISTS bus_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name VARCHAR(255) UNIQUE NOT NULL,
    image_data BYTEA NOT NULL,
    content_type VARCHAR(100) NOT NULL DEFAULT 'image/jpeg',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bus_images_file_name ON bus_images(file_name);
