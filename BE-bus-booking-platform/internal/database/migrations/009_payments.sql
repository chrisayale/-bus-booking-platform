CREATE TABLE IF NOT EXISTS payments (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID,
	reference VARCHAR(50) NOT NULL UNIQUE,
	order_number VARCHAR(100),
	provider_reference VARCHAR(100),
	amount NUMERIC(10,2) NOT NULL,
	currency VARCHAR(10) NOT NULL DEFAULT 'USD',
	payment_method VARCHAR(30) NOT NULL DEFAULT 'MOBILE_MONEY',
	status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'CANCELLED')),
	description TEXT,
	callback_data TEXT,
	metadata TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_reference ON payments(reference);
CREATE INDEX IF NOT EXISTS idx_payments_order_number ON payments(order_number);