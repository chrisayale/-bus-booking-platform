package utils

import (
	"database/sql"
	"errors"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/jmoiron/sqlx"
)

// GetCurrentExchangeRate retrieves the current USD to CDF exchange rate from the database.
func GetCurrentExchangeRate(db *sqlx.DB) (float64, error) {
	rate, err := models.GetCurrent(db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// If no rate found, return a default rate and log? For now, return 0 and let caller handle.
			return 0, err
		}
		return 0, err
	}
	return rate.USDToCDF, nil
}
