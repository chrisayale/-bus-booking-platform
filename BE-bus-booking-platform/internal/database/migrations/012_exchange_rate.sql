-- 012_exchange_rate.sql
CREATE TABLE IF NOT EXISTS exchange_rates (
    id SERIAL PRIMARY KEY,
    usd_to_cdf DECIMAL(10,2) NOT NULL DEFAULT 2800.00,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by VARCHAR(255) -- Pour suivre quel super-admin a modifié le taux
);

-- Insérer la valeur par défaut si la table est vide
INSERT INTO exchange_rates (usd_to_cdf, updated_by)
SELECT 2800.00, 'system'
WHERE NOT EXISTS (SELECT 1 FROM exchange_rates);