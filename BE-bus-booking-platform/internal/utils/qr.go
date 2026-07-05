package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type QRPayload struct {
	TicketID  string `json:"ticket_id"`
	JourneyID string `json:"journey_id"`
	Seat      string `json:"seat"`
	IssueDate string `json:"issue_date"`
	Signature string `json:"signature"`
}

// GenerateQRToken creates a signed, compact QR token using base64url encoding
// (50% shorter than hex — makes QR codes more scannable).
func GenerateQRToken(ticketID, journeyID, seat, secret string) (string, error) {
	payload := QRPayload{
		TicketID:  ticketID,
		JourneyID: journeyID,
		Seat:      seat,
		IssueDate: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal QR payload: %w", err)
	}

	// Create a signature using the secret
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(secret))
	payload.Signature = hex.EncodeToString(h.Sum(nil))

	// Re-marshal with signature
	final, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal final payload: %w", err)
	}

	// Encode to base64 URL-safe (no padding) — compact for QR codes
	return base64.RawURLEncoding.EncodeToString(final), nil
}

// DecodeQRToken extracts the journey_id from a QR token without validating the signature.
// This is used by the scanner to auto-fill the journey_id field.
// Supports both new base64 tokens and legacy hex-encoded tokens.
func DecodeQRToken(token string) (ticketID, journeyID, seat string, err error) {
	var raw []byte

	// Try base64 first (new format), fall back to hex (legacy)
	raw, err = base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		raw, err = hex.DecodeString(token)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid token encoding")
		}
	}

	var payload QRPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", "", fmt.Errorf("invalid token format")
	}

	return payload.TicketID, payload.JourneyID, payload.Seat, nil
}

func GenerateDownloadToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate download token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GenerateReference() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate booking reference: %w", err)
	}
	return "BB" + hex.EncodeToString(b)[:8], nil
}
