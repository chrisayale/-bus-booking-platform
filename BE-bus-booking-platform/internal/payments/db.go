package payments

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Statuts possibles (doivent correspondre au CHECK constraint de la migration 009_payments.sql)
const (
	StatusPending   = "PENDING"
	StatusSuccess   = "SUCCESS"
	StatusFailed    = "FAILED"
	StatusCancelled = "CANCELLED"
	StatusExpired   = "EXPIRED"
)

// Moyens de paiement (colonne payment_method)
const (
	MethodMobileMoney = "MOBILE_MONEY"
	MethodCard        = "CARD"
)

// Payment correspond à une ligne de la table `payments`.
type Payment struct {
	ID                string         `db:"id" json:"id"`
	UserID            sql.NullString `db:"user_id" json:"user_id,omitempty"`
	Reference         string         `db:"reference" json:"reference"`
	OrderNumber       sql.NullString `db:"order_number" json:"order_number,omitempty"`
	ProviderReference sql.NullString `db:"provider_reference" json:"provider_reference,omitempty"`
	Amount            float64        `db:"amount" json:"amount"`
	Currency          string         `db:"currency" json:"currency"`
	PaymentMethod     string         `db:"payment_method" json:"payment_method"`
	Status            string         `db:"status" json:"status"`
	Description       sql.NullString `db:"description" json:"description,omitempty"`
	CallbackData      sql.NullString `db:"callback_data" json:"-"`
	Metadata          sql.NullString `db:"metadata" json:"-"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

// CreatePendingPayment insère une nouvelle ligne en statut PENDING, avant même
// d'appeler FlexPay. `metadata` doit contenir le JSON sérialisé des infos
// voyage/passagers en attente (pour créer le billet seulement après succès).
func CreatePendingPayment(db *sqlx.DB, reference string, amount float64, currency, paymentMethod, description, metadata string) (*Payment, error) {
	var p Payment
	err := db.QueryRowx(`
		INSERT INTO payments (reference, amount, currency, payment_method, status, description, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, reference, order_number, provider_reference, amount, currency,
			payment_method, status, description, callback_data, metadata, created_at, updated_at`,
		reference, amount, currency, paymentMethod, StatusPending, description, metadata,
	).StructScan(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetOrderNumber enregistre l'orderNumber renvoyé par FlexPay juste après CreatePayment.
func SetOrderNumber(db *sqlx.DB, reference, orderNumber string) error {
	_, err := db.Exec(`
		UPDATE payments SET order_number = $1, updated_at = NOW() WHERE reference = $2`,
		orderNumber, reference)
	return err
}

// GetByReference retrouve un paiement par sa référence interne.
func GetByReference(db *sqlx.DB, reference string) (*Payment, error) {
	var p Payment
	err := db.Get(&p, `SELECT * FROM payments WHERE reference = $1`, reference)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByOrderNumber retrouve un paiement par l'orderNumber FlexPay (utilisé par le callback).
func GetByOrderNumber(db *sqlx.DB, orderNumber string) (*Payment, error) {
	var p Payment
	err := db.Get(&p, `SELECT * FROM payments WHERE order_number = $1`, orderNumber)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// MarkSuccess passe un paiement en SUCCESS et enregistre la référence opérateur + le
// contenu brut du callback (utile pour debug/litiges).
func MarkSuccess(db *sqlx.DB, orderNumber, providerReference, rawCallbackBody string) error {
	_, err := db.Exec(`
		UPDATE payments
		SET status = $1, provider_reference = $2, callback_data = $3, updated_at = NOW()
		WHERE order_number = $4`,
		StatusSuccess, providerReference, rawCallbackBody, orderNumber)
	return err
}

// MarkFailed passe un paiement en FAILED (le callback FlexPay a renvoyé un code != 0).
func MarkFailed(db *sqlx.DB, orderNumber, rawCallbackBody string) error {
	_, err := db.Exec(`
		UPDATE payments
		SET status = $1, callback_data = $2, updated_at = NOW()
		WHERE order_number = $3`,
		StatusFailed, rawCallbackBody, orderNumber)
	return err
}

// MarkExpired passe un paiement en EXPIRED (le délai de paiement est dépassé).
func MarkExpired(db *sqlx.DB, orderNumber, rawCallbackBody string) error {
	_, err := db.Exec(`
		UPDATE payments
		SET status = $1, callback_data = $2, updated_at = NOW()
		WHERE order_number = $3`,
		StatusExpired, rawCallbackBody, orderNumber)
	return err
}

// MarkCancelled passe un paiement en CANCELLED (le client a annulé sur la page FlexPay).
func MarkCancelled(db *sqlx.DB, orderNumber string) error {
	_, err := db.Exec(`
		UPDATE payments SET status = $1, updated_at = NOW() WHERE order_number = $2`,
		StatusCancelled, orderNumber)
	return err
}