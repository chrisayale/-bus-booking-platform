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
	Currency    string `json:"currency"` // CDF ou USD
	CallbackURL string `json:"callbackUrl"` // Respecter l'orthographe exacte
}

type CardPaymentRequest struct {
	Merchant   string `json:"merchant"`
	Type       string `json:"type"`
	Reference  string `json:"reference"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Description string `json:"description"`
	CallbackURL string `json:"callback_url"`
	ApproveURL  string `json:"approve_url"`
	CancelURL   string `json:"cancel_url"`
	DeclineURL  string `json:"decline_url"`
	Authorization string `json:"authorization,omitempty"`
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
	// Use PAY_BASE_URL for payment endpoints (mobile money and card payments)
	baseURL := os.Getenv("FLEXPAY_PAY_BASE_URL")
	if baseURL == "" {
		// Fallback to old variable for backward compatibility
		baseURL = os.Getenv("FLEXPAY_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "http://backend.flexpay.cd/api/rest/v1/paymentService"
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
	// For Mobile Money payments, use the specific HTTP endpoint
	var endpoint string
	if strings.HasSuffix(p.BaseURL, "/api/rest/v1/paymentService") {
		// If baseURL already contains the payment service path, use it but ensure it's HTTP
		endpoint = strings.Replace(p.BaseURL, "https://", "http://", 1)
	} else {
		endpoint = fmt.Sprintf("http://backend.flexpay.cd/api/rest/v1/paymentService")
	}

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
	httpReq.Header.Set("Accept", "application/json")

	authHeader := p.Authorization
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}
	httpReq.Header.Set("Authorization", authHeader)

	// Log request headers for debugging
	log.Printf("[FLEXPAY] Request Headers: Content-Type=%s, Authorization=%s",
		httpReq.Header.Get("Content-Type"),
		httpReq.Header.Get("Authorization"))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Log response status and headers for debugging
	log.Printf("[FLEXPAY] Response Status: %d", resp.StatusCode)
	log.Printf("[FLEXPAY] Response Headers: %v", resp.Header)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log the raw response body from FlexPay for debugging
	log.Printf("[FLEXPAY] Raw response body: %q", string(respBody)) // Using %q to show empty string clearly

	var result PaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("réponse FlexPay invalide: %s", string(respBody))
	}

	return &result, nil
}

// CheckOrderStatus vérifie l'état d'un paiement via l'API FlexPay
func (p *FlexPayProvider) CheckOrderStatus(orderNumber string) (*CheckPaymentResponse, error) {
	// For check requests, use the dedicated backend endpoint for status checks
	endpoint := fmt.Sprintf("https://backend.flexpay.cd/api/rest/v1/check/%s", orderNumber)

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	authHeader := p.Authorization
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}
	httpReq.Header.Set("Authorization", authHeader)

	// Log request headers for debugging
	log.Printf("[FLEXPAY] Request Headers: Content-Type=%s, Authorization=%s",
		httpReq.Header.Get("Content-Type"),
		httpReq.Header.Get("Authorization"))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Log response status and headers for debugging
	log.Printf("[FLEXPAY] Response Status: %d", resp.StatusCode)
	log.Printf("[FLEXPAY] Response Headers: %v", resp.Header)

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
	log.Printf("[FLEXPAY] Raw response body for CheckOrderStatus request (orderNumber=%s): %q", orderNumber, string(respBody)) // Using %q to show empty string clearly

	var result CheckPaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("réponse FlexPay invalide: %s", string(respBody))
	}
	if result.Transaction == nil {
		return nil, fmt.Errorf("FlexPay response missing transaction: %s", string(respBody))
	}

	return &result, nil
}

// CreateCardPayment effectue un paiement par carte bancaire via l'API FlexPay
func (p *FlexPayProvider) CreateCardPayment(req CardPaymentRequest) (*PaymentResponse, error) {
	// Use dedicated card payment endpoint with environment variable override
	cardEndpoint := os.Getenv("FLEXPAY_CARD_ENDPOINT")
	if cardEndpoint == "" {
		cardEndpoint = "https://cardpayment.flexpay.cd/v1.1/pay"
	}

	req.Merchant = p.Merchant
	req.Type = "2" // Forcer le type carte bancaire
	req.Description = "Paiement billet de voyage GUV"

	// Set authorization field for V2 card endpoint authentication (Bearer token in JSON body)
	authValue := os.Getenv("FLEXPAY_TOKEN")
	if authValue == "" {
		// Fallback to using authorization header value
		authValue = p.Authorization
	}
	// Ensure it has the Bearer prefix if not already present
	if authValue != "" && !strings.HasPrefix(strings.ToLower(authValue), "bearer ") {
		authValue = "Bearer " + authValue
	}
	req.Authorization = authValue

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Log the exact URL and JSON payload being sent to FlexPay
	log.Printf("[FLEXPAY] Calling URL: %s", cardEndpoint)
	log.Printf("[FLEXPAY] JSON payload: %s", string(data))

	httpReq, err := http.NewRequest("POST", cardEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	// Note: Authorization is now in the JSON body, not in headers for V2 card endpoint

	// Log request headers for debugging
	log.Printf("[FLEXPAY] Request Headers: Content-Type=%s, Accept=%s",
		httpReq.Header.Get("Content-Type"),
		httpReq.Header.Get("Accept"))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Log response status and headers for debugging
	log.Printf("[FLEXPAY] Response Status: %d", resp.StatusCode)
	log.Printf("[FLEXPAY] Response Headers: %v", resp.Header)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log the raw response body from FlexPay for debugging
	log.Printf("[FLEXPAY] Raw response body: %q", string(respBody)) // Using %q to show empty string clearly

	var result PaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("réponse FlexPay invalide: %s", string(respBody))
	}

	return &result, nil
}
