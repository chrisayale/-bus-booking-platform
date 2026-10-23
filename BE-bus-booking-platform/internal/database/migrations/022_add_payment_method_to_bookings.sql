-- Add payment_method column to bookings table
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS payment_method VARCHAR(30)
    CHECK (payment_method IN ('cash', 'mobile_money', 'card'));

-- Create index for controller queries
CREATE INDEX IF NOT EXISTS idx_bookings_payment_method ON bookings(payment_method);