package service

import (
	"context"
	"fmt"

	"digital-shop/internal/config"

	"github.com/plutov/paypal/v4"
)

type PayPalService struct {
	client    *paypal.Client
	webhookID string
}

func NewPayPalService(cfg *config.Config) (*PayPalService, error) {
	base := paypal.APIBaseLive
	if cfg.PayPalSandbox {
		base = paypal.APIBaseSandBox
	}
	c, err := paypal.NewClient(cfg.PayPalClientID, cfg.PayPalSecret, base)
	if err != nil {
		return nil, err
	}
	return &PayPalService{client: c, webhookID: cfg.PayPalWebhookID}, nil
}

func (s *PayPalService) CreateOrder(ctx context.Context, internalOrderID, currency string, totalCents int64) (string, error) {
	amount := paypal.PurchaseUnitAmount{
		Currency: currency,
		Value:    fmt.Sprintf("%.2f", float64(totalCents)/100.0),
	}
	pu := paypal.PurchaseUnitRequest{
		Amount:      &amount,
		CustomID:    internalOrderID,
		Description: "Digital product order " + internalOrderID,
	}
	order, err := s.client.CreateOrder(ctx, paypal.OrderIntentCapture, []paypal.PurchaseUnitRequest{pu}, nil, nil)
	if err != nil {
		return "", err
	}
	return order.ID, nil
}

func (s *PayPalService) CaptureOrder(ctx context.Context, paypalOrderID string) (captureID string, err error) {
	resp, err := s.client.CaptureOrder(ctx, paypalOrderID, paypal.CaptureOrderParams{})
	if err != nil {
		return "", err
	}
	for _, unit := range resp.PurchaseUnits {
		for _, cap := range unit.Payments.Captures {
			if cap.Status == "COMPLETED" {
				return cap.ID, nil
			}
		}
	}
	return "", fmt.Errorf("no completed capture found in paypal response")
}

func (s *PayPalService) RefundCapture(ctx context.Context, captureID, currency string, amountCents int64) (string, error) {
	req := paypal.RefundCaptureRequest{
		Amount: &paypal.Money{
			Currency: currency,
			Value:    fmt.Sprintf("%.2f", float64(amountCents)/100.0),
		},
	}
	r, err := s.client.RefundCapture(ctx, captureID, req)
	if err != nil {
		return "", err
	}
	return r.ID, nil
}

func (s *PayPalService) VerifyWebhook(ctx context.Context, req *http.Request) error {
	resp, err := s.client.VerifyWebhookSignature(ctx, req, s.webhookID)
	if err != nil {
		return err
	}
	if resp.VerificationStatus != "SUCCESS" {
		return fmt.Errorf("webhook verification failed: %s", resp.VerificationStatus)
	}
	return nil
}
