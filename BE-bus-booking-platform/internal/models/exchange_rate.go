package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// ExchangeRate représente le taux de change USD vers CDF
type ExchangeRate struct {
	ID        int64     `db:"id"`
	USDToCDF  float64   `db:"usd_to_cdf"`
	UpdatedAt time.Time `db:"updated_at"`
	UpdatedBy string    `db:"updated_by"`
}

// TableName retourne le nom de la table (optionnel, mais utile pour certaines pratiques)
func (ExchangeRate) TableName() string {
	return "exchange_rates"
}

// GetCurrent récupère le taux de change actuel depuis la base de données
func GetCurrent(db *sqlx.DB) (*ExchangeRate, error) {
	var rate ExchangeRate
	err := db.Get(&rate, "SELECT * FROM exchange_rates ORDER BY id DESC LIMIT 1")
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

// Update met à jour le taux de change
func (r *ExchangeRate) Update(db *sqlx.DB, newRate float64, updatedBy string) error {
	r.USDToCDF = newRate
	r.UpdatedAt = time.Now()
	r.UpdatedBy = updatedBy

	_, err := db.NamedExec(`
		UPDATE exchange_rates
		SET usd_to_cdf = :usd_to_cdf,
		    updated_at = :updated_at,
		    updated_by = :updated_by
		WHERE id = :id`, r)
	return err
}
