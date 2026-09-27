package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/bus-booking/BE-bus-booking-platform/internal/payments"
	"github.com/bus-booking/BE-bus-booking-platform/internal/whatsapp"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.mau.fi/whatsmeow/types"
)

type PaymentHandler struct {
	DB *sqlx.DB
}

func NewPaymentHandler(db *sqlx.DB) *PaymentHandler {
	return &PaymentHandler{DB: db}
}

type flexPayCallbackPayload struct {
	Code              string `json:"code"`
	Reference         string `json:"reference"`
	ProviderReference string `json:"provider_reference"`
	OrderNumber       string `json:"orderNumber"`
	Message           string `json:"message,omitempty"`
}

// FlexPayCallback reçoit la notification serveur-à-serveur envoyée par FlexPay
// (callback_url). C'est la SEULE source de vérité sur le statut réel du paiement —
// approve_url/cancel_url/decline_url ne sont que des redirections visuelles pour le
// client et ne doivent jamais être utilisées pour confirmer un paiement.
func (h *PaymentHandler) FlexPayCallback(c *gin.Context) {
	log.Printf("payment callback: received payload")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var payload flexPayCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	log.Printf("payment callback: payload: %+v", payload)

	if payload.Code == "0" {
		// Idempotency check: verify payment status before processing
		existingPayment, err := payments.GetByOrderNumber(h.DB, payload.OrderNumber)
		if err == nil {
			switch existingPayment.Status {
			case payments.StatusSuccess, payments.StatusFailed:
				// Payment already processed, return success to prevent webhook retries
				log.Printf("payment callback: payment %s already processed with status %s, skipping", payload.OrderNumber, existingPayment.Status)
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
				return
			}
		}

		if err := payments.MarkSuccess(h.DB, payload.OrderNumber, payload.ProviderReference, string(body)); err != nil {
			log.Printf("payment callback: failed to mark success: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
			return
		}

		p, err := payments.GetByOrderNumber(h.DB, payload.OrderNumber)
		if err != nil {
			log.Printf("payment callback: payment not found for order %s: %v", payload.OrderNumber, err)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}

		if err := whatsapp.CompleteBookingFromPayment(p); err != nil {
			log.Printf("payment callback: failed to complete booking for %s: %v", p.Reference, err)
		}

		// Explicit success response to prevent FlexPay webhook retry loops
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	} else {
		// Handle payment failure - check for specific error types and notify user via WhatsApp
		if err := payments.MarkFailed(h.DB, payload.OrderNumber, string(body)); err != nil {
			log.Printf("payment callback: failed to mark failed: %v", err)
		}

		// Get payment details to extract user information for WhatsApp notification - FIXED JID HANDLING
		payment, err := payments.GetByOrderNumber(h.DB, payload.OrderNumber)
		if err == nil && payment.Metadata.Valid && payment.Metadata.String != "" {
			var pendingMeta whatsapp.PendingBookingMetadata
			if json.Unmarshal([]byte(payment.Metadata.String), &pendingMeta) == nil {
				var jid types.JID
				if pendingMeta.RecipientJID != "" {
					var parseErr error
					jid, parseErr = types.ParseJID(pendingMeta.RecipientJID)
					if parseErr != nil {
						// If parsing fails, clear jid to fall back to phone number
						jid = types.JID{}
					}
				}
				if jid.User == "" {
					// Fallback: extract and clean the guest phone number (for backward compatibility)
					cleanedPhone := whatsapp.CleanPhoneNumber(pendingMeta.GuestPhone)
					if cleanedPhone != "" {
						jid = types.NewJID(cleanedPhone, types.DefaultUserServer)
					}
				}

				// Only proceed if we have a valid JID
				if jid.User != "" {
					// Check for specific error patterns in the FlexPay response
					errorMessage := strings.ToLower(payload.Message)
					isPinError := strings.Contains(errorMessage, "pin") ||
						strings.Contains(errorMessage, "incorrect") ||
						strings.Contains(errorMessage, "invalid") ||
						strings.Contains(errorMessage, "authentication")

					isTransactionRefused := strings.Contains(errorMessage, "refused") ||
						strings.Contains(errorMessage, "declined") ||
						strings.Contains(errorMessage, "not authorized") ||
						strings.Contains(errorMessage, "transaction failed")

					var whatsappMessage string
					if isPinError {
						whatsappMessage = "🔒 Votre code PIN est incorrect.\n\nChoisissez une option :\n1️⃣ Relancer le paiement\n2️⃣ Annuler / Retour au menu principal"
					} else if isTransactionRefused {
						whatsappMessage = "💳 Votre transaction a été refusée par la banque ou l'opérateur.\n\nChoisissez une option :\n1️⃣ Relancer le paiement\n2️⃣ Annuler / Retour au menu principal"
					} else {
						// Generic failure message
						whatsappMessage = "😔 Votre paiement a échoué.\n\nChoisissez une option :\n1️⃣ Relancer le paiement\n2️⃣ Annuler / Retour au menu principal"
					}

					// Send WhatsApp notification to user
					whatsapp.SendText(jid, whatsappMessage)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}