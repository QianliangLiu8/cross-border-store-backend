package service

import (
	"context"
	"errors"
	"time"

	"digital-shop/internal/model"
	"digital-shop/internal/statemachine"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RefundService struct {
	db     *gorm.DB
	paypal *PayPalService
}

func NewRefundService(db *gorm.DB, pp *PayPalService) *RefundService {
	return &RefundService{db: db, paypal: pp}
}

// CreatePending 创建人工待审核的退款申请（可由前端或客服触发）。
func (s *RefundService) CreatePending(ctx context.Context, orderID string) (*model.Refund, error) {
	var order model.Order
	if err := s.db.Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	if order.Status != statemachine.StatePaid &&
		order.Status != statemachine.StateDelivered &&
		order.Status != statemachine.StateCompleted {
		return nil, errors.New("order status does not allow refund")
	}
	r := &model.Refund{
		OrderID:     orderID,
		AmountCents: order.TotalCents,
		Status:      "PENDING",
	}
	if err := s.db.Create(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

// Approve 管理员审核通过 → 调 PayPal 全额原路退款。
func (s *RefundService) Approve(ctx context.Context, refundID uint, adminID string) (*model.Refund, error) {
	var refund model.Refund
	var order model.Order

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", refundID).First(&refund).Error; err != nil {
			return err
		}
		if refund.Status != "PENDING" {
			return errors.New("refund is not pending")
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", refund.OrderID).First(&order).Error; err != nil {
			return err
		}

		next, err := statemachine.Transition(order.Status, statemachine.EventRefundApproved)
		if err != nil {
			return err
		}
		if order.PayPalCaptureID == "" {
			return errors.New("order has no paypal capture id")
		}

		ppRefundID, err := s.paypal.RefundCapture(ctx, order.PayPalCaptureID, order.Currency, refund.AmountCents)
		if err != nil {
			// 保持 REFUNDING，允许重试
			_ = tx.Model(&order).Update("status", next).Error
			return err
		}

		final, err := statemachine.Transition(next, statemachine.EventRefundCompleted)
		if err != nil {
			return err
		}

		now := time.Now()
		if err := tx.Model(&order).Updates(map[string]any{
			"status":     final,
			"closed_at":  now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&refund).Updates(map[string]any{
			"status":           "COMPLETED",
			"admin_id":         adminID,
			"paypal_refund_id": ppRefundID,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	s.db.Where("id = ?", refundID).First(&refund)
	return &refund, nil
}

// Reject 管理员拒绝退款（不调用 PayPal）。
func (s *RefundService) Reject(ctx context.Context, refundID uint, adminID, reason string) (*model.Refund, error) {
	var refund model.Refund
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", refundID).First(&refund).Error; err != nil {
			return err
		}
		if refund.Status != "PENDING" {
			return errors.New("refund is not pending")
		}
		return tx.Model(&refund).Updates(map[string]any{
			"status":        "REJECTED",
			"admin_id":      adminID,
			"reject_reason": reason,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	s.db.Where("id = ?", refundID).First(&refund)
	return &refund, nil
}
