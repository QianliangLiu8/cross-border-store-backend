package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TaxJarService struct {
	apiKey string
	client *http.Client
}

type TaxJarLineItem struct {
	ID             string  `json:"id"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	ProductTaxCode string  `json:"product_tax_code,omitempty"`
}

type TaxJarRequest struct {
	FromCountry string           `json:"from_country"`
	FromZip     string           `json:"from_zip,omitempty"`
	FromState   string           `json:"from_state,omitempty"`
	ToCountry   string           `json:"to_country"`
	ToZip       string           `json:"to_zip,omitempty"`
	ToState     string           `json:"to_state,omitempty"`
	Amount      float64          `json:"amount"`
	Shipping    float64          `json:"shipping"`
	LineItems   []TaxJarLineItem `json:"line_items"`
}

type TaxJarResponse struct {
	Tax struct {
		AmountToCollect float64 `json:"amount_to_collect"`
		Rate            float64 `json:"rate"`
		Jurisdictions   struct {
			Country string `json:"country"`
			State   string `json:"state"`
			City    string `json:"city"`
		} `json:"jurisdictions"`
	} `json:"tax"`
}

func NewTaxJarService(apiKey string) *TaxJarService {
	return &TaxJarService{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *TaxJarService) Calculate(ctx context.Context, req TaxJarRequest) (*TaxJarResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.taxjar.com/v2/taxes", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("taxjar returned status %d", resp.StatusCode)
	}

	var out TaxJarResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
