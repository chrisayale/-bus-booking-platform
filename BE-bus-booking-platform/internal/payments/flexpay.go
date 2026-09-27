package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type FlexPayProvider struct {
	BaseURL       string
	Authorization string
	Merchant      string
	httpClient    *http.Client
}

type MobilePaymentRequest struct {
	Merchant    string `json:"merchant"`
	Type        string `json:"type"` // Doit être "1" pour Mobile Money
	Reference   string `json:"reference"`
	Phone       string `json:"phone"` // Ex: 243891234567
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`    // CDF ou USD
	Description string `json:"description"`
	CallbackURL string `json:"callbackUrl"` // Respecter l'orthographe exacte
}

type CardPaymentRequest struct {
	Merchant    string `json:"merchant"`
	Type        string `json:"type"`
	Reference   string `json:"reference"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	CallbackURL string `json:"callbackUrl"`
	ApproveURL  string `json:"approveUrl"`
	CancelURL   string `json:"cancelUrl"`
	DeclineURL  string `json:"declineUrl"`
}

type PaymentResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	URL         string `json:"url,omitempty"`
	OrderNumber string `json:"orderNumber,omitempty"`
}

type CheckPaymentResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Transaction *struct {
		Reference      string `json:"reference"`
		OrderNumber    string `json:"orderNumber"`
		Status         string `json:"status"` // 0: Succès, 1: Échec, 2: En attente, 3: Non trouvé
		Amount         string `json:"amount"`
		AmountCustomer string `json:"amountCustomer"`
		Currency       string `json:"currency"`
		CreatedAt      string `json:"createdAt"`
		Channel        string `json:"channel"`
	} `json:"transaction"`
}

func NewFlexPayProvider() *FlexPayProvider {
	baseURL := os.Getenv("FLEXPAY_BASE_URL")
	if baseURL == "" {
		baseURL = "https://backend.flexpay.cd"
	}

	auth := os.Getenv("FLEXPAY_AUTHORIZATION")
	merchant := os.Getenv("FLEXPAY_MERCHANT")
	if merchant == "" {
		merchant = "KIVUGC"
	}

	// DIAGNOSTIC DU TOKEN : Affiche des infos dans les logs pour analyser le format
	if auth == "" {
		log.Println("[FLEXPAY] ATTENTION: La variable FLEXPAY_AUTHORIZATION est vide !")
	} else {
		parts := strings.Split(auth, ".")
		maskedAuth := auth
		if len(auth) > 10 {
			maskedAuth = auth[:5] + "..." + auth[len(auth)-5:]
		}
		log.Printf("[FLEXPAY] Token chargé -> Longueur: %d | Nombre de parties (séparées par des points): %d | Valeur masquée: %s\n", 
			len(auth), len(parts), maskedAuth)
	}

	return &FlexPayProvider{
		BaseURL:       baseURL,
		Authorization: auth,
		Merchant:      merchant,
		httpClient:    &http.Client{Timeout: 300 * time.Second},
	}
}

// 1. Paiement Mobile Money (Push USSD)
func (p *FlexPayProvider) CreateMobilePayment(req MobilePaymentRequest) (*PaymentResponse, error) {
	endpoint := fmt.Sprintf("%s/api/rest/v1/paymentService", p.BaseURL)

	req.Merchant = p.Merchant
	req.Type = "1" // Forcer le type Mobile Money

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Log the exact URL and JSON payload being sent to FlexPay
	log.Printf("[FLEXPAY] Calling URL: %s", endpoint)
	log.Printf("[FLEXPAY] JSON payload: %s", string(data))

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	authHeader := p.Authorization
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log the raw response body from FlexPay for debugging
	log.Printf("[FLEXPAY] Raw response body: %s", string(respBody))

	var result PaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("réponse FlexPay invalide: %s", string(respBody))
	}

	return &result, nil
}

// CheckOrderStatus vérifie l'état d'un paiement via l'API FlexPay
func (p *FlexPayProvider) CheckOrderStatus(orderNumber string) (*CheckPaymentResponse, error) {
	endpoint := fmt.Sprintf("%s/api/rest/v1/check/%s", p.BaseURL, orderNumber)

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	authHeader := p.Authorization
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Return a synthetic response indicating transaction still pending (status 2)
		// Treat 404 as "transaction not found yet / still pending" to avoid logging hard errors
		return &CheckPaymentResponse{
			Transaction: &struct {
				Reference      string `json:"reference"`
				OrderNumber    string `json:"orderNumber"`
				Status         string `json:"status"` // 0: Succès, 1: Échec, 2: En attente, 3: Non trouvé
				Amount         string `json:"amount"`
				AmountCustomer string `json:"amountCustomer"`
				Currency       string `json:"currency"`
				CreatedAt      string `json:"createdAt"`
				Channel        string `json:"channel"`
			}{
				Status:      "2", // En attente (pending)
				OrderNumber: orderNumber,
			},
		}, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log the raw response body from FlexPay for debugging
	log.Printf("[FLEXPAY] Raw response body for CheckOrderStatus request (orderNumber=%s): %s", orderNumber, string(respBody))

	var result CheckPaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("réponse FlexPay invalide: %s", string(respBody))
	}
	if result.Transaction == nil {
		return nil, fmt.Errorf("FlexPay response missing transaction: %s", string(respBody))
	}

	return &result, nil
}

