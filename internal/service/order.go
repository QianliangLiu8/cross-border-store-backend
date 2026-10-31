package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"digital-shop/internal/config"
	"digital-shop/internal/model"
	"digital-shop/internal/statemachine"
	"digital-shop/internal/util"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderService struct {
	db     *gorm.DB
	cfg    *config.Config
	paypal *PayPalService
	taxjar *TaxJarService
}

func NewOrderService(db *gorm.DB, cfg *config.Config, pp *PayPalService, tj *TaxJarService) *OrderService {
	return &OrderService{db: db, cfg: cfg, paypal: pp, taxjar: tj}
}

type CreateItem struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
}

type CreateAddress struct {
	Country string `json:"country" binding:"required"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
}

type CreateOrderReq struct {
	UserID  string        `json:"user_id" binding:"required"`
	Items   []CreateItem  `json:"items" binding:"required,dive"`
	Address CreateAddress `json:"address" binding:"required"`
}

type CreateOrderResp struct {
	Order         *model.Order `json:"order"`
	PayPalOrderID string       `json:"paypal_order_id"`
}

func (s *OrderService) Create(ctx context.Context, req CreateOrderReq) (*CreateOrderResp, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("empty items")
	}

	skus := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		skus = append(skus, it.SKU)
	}
	var variants []model.ProductVariant
	if err := s.db.Where("sku IN ?", skus).Find(&variants).Error; err != nil {
		return nil, err
	}
	index := make(map[string]*model.ProductVariant, len(variants))
	for i := range variants {
		index[variants[i].SKU] = &variants[i]
	}

	var subtotal int64
	items := make([]model.OrderItem, 0, len(req.Items))
	lines := make([]TaxJarLineItem, 0, len(req.Items))
	for _, in := range req.Items {
		v, ok := index[in.SKU]
		if !ok {
			return nil, fmt.Errorf("sku not found: %s", in.SKU)
		}
		lineTotal := v.PriceCents * int64(in.Quantity)
		subtotal += lineTotal
		items = append(items, model.OrderItem{
			ProductID:      v.ProductID,
			VariantSKU:     v.SKU,
			ProductName:    v.VariantName,
			VariantName:    v.VariantName,
			UnitPriceCents: v.PriceCents,
			Quantity:       in.Quantity,
		})
		lines = append(lines, TaxJarLineItem{
			ID:        v.SKU,
			Quantity:  in.Quantity,
			UnitPrice: float64(v.PriceCents) / 100.0,
		})
	}

	// 1. 调 TaxJar 算税
	taxResp, err := s.taxjar.Calculate(ctx, TaxJarRequest{
		FromCountry: "US",
		ToCountry:   req.Address.Country,
		ToZip:       req.Address.Zip,
		ToState:     req.Address.State,
		Amount:      float64(subtotal) / 100.0,
		Shipping:    0,
		LineItems:   lines,
	})
	if err != nil {
		return nil, fmt.Errorf("tax calculation failed: %w", err)
	}
	taxCents := int64(taxResp.Tax.AmountToCollect*100 + 0.5)
	jurisdiction := taxResp.Tax.Jurisdictions.Country + "-" + taxResp.Tax.Jurisdictions.State

	// 2. 落库订单（PENDING_PAYMENT）
	order := &model.Order{
		ID:              "ORD" + uuid.NewString(),
		UserID:          req.UserID,
		Status:          statemachine.StatePendingPayment,
		Currency:        "USD",
		SubtotalCents:   subtotal,
		TaxCents:        taxCents,
		TotalCents:      subtotal + taxCents,
		TaxRate:         taxResp.Tax.Rate,
		TaxJurisdiction: jurisdiction,
		ToCountry:       req.Address.Country,
		ToState:         req.Address.State,
		ToZip:           req.Address.Zip,
		Items:           items,
	}
	if err := s.db.Create(order).Error; err != nil {
		return nil, err
	}

	// 3. 创建 PayPal 订单
	ppID, err := s.paypal.CreateOrder(ctx, order.ID, order.Currency, order.TotalCents)
	if err != nil {
		return nil, fmt.Errorf("paypal create order failed: %w", err)
	}
	if err := s.db.Model(order).Update("paypal_order_id", ppID).Error; err != nil {
		return nil, err
	}
	order.PayPalOrderID = ppID

	return &CreateOrderResp{Order: order, PayPalOrderID: ppID}, nil
}

// Capture 用于前端从 PayPal 跳回后主动确认捕获。
func (s *OrderService) Capture(ctx context.Context, internalOrderID string) (*model.Order, error) {
	var order model.Order
	if err := s.db.Preload("Items").Where("id = ?", internalOrderID).First(&order).Error; err != nil {
		return nil, err
	}
	if order.Status != statemachine.StatePendingPayment {
		return &order, nil
	}

	captureID, err := s.paypal.CaptureOrder(ctx, order.PayPalOrderID)
	if err != nil {
		return nil, err
	}
	return s.applyPaidState(ctx, order.ID, captureID)
}

// applyPaidState 幂等应用「支付成功」事件：状态机迁移 + 虚拟发货。
func (s *OrderService) applyPaidState(ctx context.Context, orderID, captureID string) (*model.Order, error) {
	var order model.Order
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		if order.Status != statemachine.StatePendingPayment {
			return nil // 已经处理过，直接幂等返回
		}
		next, err := statemachine.Transition(order.Status, statemachine.EventPaymentCaptured)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&order).Updates(map[string]any{
			"status":            next,
			"paypal_capture_id": captureID,
			"paid_at":           now,
		}).Error; err != nil {
			return err
		}
		order.Status = next
		order.PayPalCaptureID = captureID
		order.PaidAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// MarkDelivered 把订单标记为已发货（虚拟商品发货由支付成功自动触发）。
func (s *OrderService) MarkDelivered(ctx context.Context, orderID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		if order.Status != statemachine.StatePaid {
			return nil
		}
		next, err := statemachine.Transition(order.Status, statemachine.EventDigitalDelivered)
		if err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&order).Updates(map[string]any{
			"status":       next,
			"delivered_at": now,
		}).Error
	})
}

// BuildDownloadURLs 为已支付订单生成带签名的下载链接。
func (s *OrderService) BuildDownloadURLs(order *model.Order) []map[string]any {
	links := make([]map[string]any, 0, len(order.Items))
	for _, it := range order.Items {
		links = append(links, map[string]any{
			"sku":          it.VariantSKU,
			"variant_name": it.VariantName,
			"url": util.GenerateDownloadURL(
				s.cfg.DownloadBaseURL, s.cfg.DownloadSecret,
				order.ID, it.VariantSKU, s.cfg.DownloadTTL,
			),
			"expires_in": s.cfg.DownloadTTL.String(),
		})
	}
	return links
}
